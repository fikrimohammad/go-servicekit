package conf

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fikrimohammad/go-servicekit/observability/logs"
	"github.com/fikrimohammad/go-servicekit/observability/metrics"
	"github.com/fikrimohammad/go-servicekit/observability/tracer"
	"go.opentelemetry.io/otel/attribute"
)

// Provider is the storage contract. A Loader binds one Provider.
type Provider interface {
	// Source returns a stable, low-cardinality telemetry label: "file" or
	// "etcd" for the built-in providers.
	Source() string
	// Load returns the raw value for each requested key. Missing keys are
	// absent from the map. rev is opaque to the Loader and is passed back to
	// Watch. Load is called once per Loader.
	Load(ctx context.Context, keys []string) (map[string]string, int64, error)
	// Close releases resources. It is idempotent and safe before a successful
	// Load.
	Close() error
}

// Watcher is a Provider that streams changes.
type Watcher interface {
	// Watch streams changes after rev. The channel closes only when ctx is
	// done. The provider owns reconnect and resync.
	Watch(ctx context.Context, rev int64) (<-chan Event, error)
}

// Formatter is a Provider that owns parsing for its values. The loader calls
// Parse for every Load result and rejects per-leaf format tags. The file
// provider implements it; the etcd provider does not.
type Formatter interface {
	// Parse decodes raw into a value of type t.
	Parse(raw string, t reflect.Type) (any, error)
}

// getterLeaf is implemented by *Getter[T]. It lets the loader recognize,
// bind, and type a leaf without matching type names.
type getterLeaf interface {
	bind(core func() (any, error))
	target() reflect.Type
}

type options struct {
	metrics metrics.Client
	tracer  tracer.Client
	logger  logs.Logger
}

// Option configures a Loader at construction time.
type Option func(*options) error

// WithMetrics injects a metrics client. A nil client falls back to
// metrics.Default, read per operation.
func WithMetrics(m metrics.Client) Option {
	return func(o *options) error {
		o.metrics = m
		return nil
	}
}

// WithTracer injects a tracer client. A nil client falls back to
// tracer.Default, read per operation.
func WithTracer(t tracer.Client) Option {
	return func(o *options) error {
		o.tracer = t
		return nil
	}
}

// WithLogger injects a logger. A nil logger falls back to logs.Default, read
// per operation.
func WithLogger(l logs.Logger) Option {
	return func(o *options) error {
		o.logger = l
		return nil
	}
}

type snapshot struct {
	values map[string]any
}

// Loader binds one Provider to a struct of Getter fields.
type Loader[T any] struct {
	*loaderCore
	data T
}

type loaderCore struct {
	provider  Provider
	formatter Formatter
	source    string
	metrics   metrics.Client
	tracer    tracer.Client
	logger    logs.Logger
	//nolint:containedctx // Close has no ctx; the detached context carries caller values.
	logCtx context.Context

	snapshot atomic.Pointer[snapshot]

	leaves map[string]leaf

	stateMu          sync.Mutex
	state            State
	degraded         map[string]struct{}
	providerDegraded bool

	closed    atomic.Bool
	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

// New builds a Loader for T, loads the provider, and binds every Getter. T
// must be a struct whose leaves are Getter fields.
func New[T any](ctx context.Context, p Provider, opts ...Option) (loader *Loader[T], err error) {
	var zero T
	tType := reflect.TypeFor[T]()
	if tType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: T must be a struct, got %T", ErrInvalidType, zero)
	}
	if p == nil {
		return nil, ErrInvalidConfig
	}

	o := options{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&o); err != nil {
			return nil, err
		}
	}

	source := p.Source()
	mc := resolveMetrics(o.metrics)
	tc := resolveTracer(o.tracer)
	lg := resolveLogger(o.logger)
	start := time.Now()
	ctx, span := startLoadSpan(ctx, tc, source)
	defer func() {
		endLoadSpan(span, err)
		recordMetric(ctx, mc, source, start, err, errorClass(err))
	}()

	formatter, ownsFormat := p.(Formatter)
	leaves, err := derive(tType, ownsFormat)
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	if len(leaves) == 0 {
		_ = p.Close()
		return nil, fmt.Errorf("%w: %s has no Getter fields", ErrInvalidType, tType)
	}

	keys := make([]string, len(leaves))
	for i, l := range leaves {
		keys[i] = l.key
	}

	raw, rev, err := p.Load(ctx, keys)
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	if rev != 0 {
		span.SetAttributes(attribute.Int64("conf.revision", rev))
	}

	var missing []error
	for _, l := range leaves {
		if _, ok := raw[l.key]; !ok {
			missing = append(missing, fmt.Errorf("%w: %s", ErrMissingKey, l.key))
		}
	}
	if len(missing) > 0 {
		_ = p.Close()
		return nil, errors.Join(missing...)
	}

	parse := func(raw string, l leaf) (any, error) {
		if formatter != nil {
			return formatter.Parse(raw, l.typ)
		}
		return parseValue(raw, l.typ, l.format)
	}

	values := make(map[string]any, len(leaves))
	for _, l := range leaves {
		v, err := parse(raw[l.key], l)
		if err != nil {
			_ = p.Close()
			return nil, fmt.Errorf("%w: key %q: %w", ErrParse, l.key, err)
		}
		values[l.key] = v
	}

	leafMap := make(map[string]leaf, len(leaves))
	for _, l := range leaves {
		leafMap[l.key] = l
	}

	core := &loaderCore{
		provider:  p,
		formatter: formatter,
		source:    source,
		metrics:   o.metrics,
		tracer:    o.tracer,
		logger:    o.logger,
		logCtx:    context.WithoutCancel(ctx),
		leaves:    leafMap,
		degraded:  make(map[string]struct{}),
		state:     StateLive,
	}
	core.snapshot.Store(&snapshot{values: values})

	dataVal := reflect.New(tType).Elem()
	for _, l := range leaves {
		key := l.key
		fv := dataVal
		for _, idx := range l.index {
			fv = fv.Field(idx)
		}
		binder, ok := reflect.TypeAssert[getterLeaf](fv.Addr())
		if !ok {
			_ = p.Close()
			return nil, fmt.Errorf("%w: field %s cannot be bound", ErrInvalidType, l.field)
		}
		binder.bind(func() (any, error) { return core.get(key) })
	}

	loader = &Loader[T]{loaderCore: core, data: dataVal.Interface().(T)}

	if w, ok := p.(Watcher); ok {
		watchCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		ch, err := w.Watch(watchCtx, rev)
		if err != nil {
			cancel()
			_ = p.Close()
			return nil, err
		}
		core.cancel = cancel
		core.done = make(chan struct{})
		go core.consume(watchCtx, ch)
	}

	lg.Info(ctx, "conf: loaded", "conf.source", source, "conf.revision", rev)
	return loader, nil
}

// Data returns the config struct with every Getter bound.
func (l *Loader[T]) Data() T { return l.data }

func (c *loaderCore) parse(raw string, l leaf) (any, error) {
	if c.formatter != nil {
		return c.formatter.Parse(raw, l.typ)
	}
	return parseValue(raw, l.typ, l.format)
}

func (c *loaderCore) get(key string) (any, error) {
	if c.closed.Load() {
		return nil, ErrClosed
	}
	s := c.snapshot.Load()
	if s == nil {
		return nil, ErrNotSet
	}
	v, ok := s.values[key]
	if !ok {
		return nil, ErrNotSet
	}
	return v, nil
}

// State reports the loader's health and lifecycle state.
func (c *loaderCore) State() State {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state
}

func (c *loaderCore) setState(s State) {
	c.stateMu.Lock()
	c.state = s
	c.stateMu.Unlock()
}

// Close stops the watch and closes the provider. It is idempotent.
func (c *loaderCore) Close() error {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.setState(StateClosed)
		if c.cancel != nil {
			c.cancel()
		}
		if c.done != nil {
			<-c.done
		}
		c.closeErr = c.provider.Close()
		if c.closeErr != nil {
			c.loggerClient().Error(c.logCtx, "conf: close failed", "error.type", "_OTHER")
		}
	})
	return c.closeErr
}
