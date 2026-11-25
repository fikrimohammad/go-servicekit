package conf

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"
	"github.com/fikrimohammad/go-servicekit/observability/logs"
	"github.com/fikrimohammad/go-servicekit/observability/metrics"
	"github.com/fikrimohammad/go-servicekit/observability/tracer"
	"go.opentelemetry.io/otel/attribute"
)

// getterLeaf is implemented by *Getter[T]. It lets the loader recognize,
// bind, and type a leaf without matching type names.
type getterLeaf interface {
	bind(core func() (any, error))
	target() reflect.Type
}

// isNilProvider reports whether p is nil or a typed nil that would panic when
// its methods run.
func isNilProvider(p confcore.Provider) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
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

// Loader binds one Source to a struct of Getter fields. Create it with [New];
// the zero value is not usable.
type Loader[T any] struct {
	core *loaderCore
	data T
}

type loaderCore struct {
	provider  confcore.Provider
	formatter confcore.Formatter
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

// New builds a Loader for T, loads the source, and binds every Getter. T must
// be a struct whose leaves are Getter fields. Create a Source with [File] or
// [Etcd].
//
// New takes ownership of the provider created by source. It closes that
// provider when it returns an error, and [Loader.Close] closes it on success.
// An etcd source is watched until [Loader.Close]: canceling ctx stops the boot
// load but does not stop the watch.
func New[T any](ctx context.Context, source Source, opts ...Option) (loader *Loader[T], err error) {
	if source.err != nil {
		return nil, source.err
	}
	if source.newProvider == nil {
		return nil, ErrInvalidConfig
	}

	var zero T
	tType := reflect.TypeFor[T]()
	if tType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: T must be a struct, got %T", ErrInvalidType, zero)
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

	p := source.newProvider()
	if isNilProvider(p) {
		return nil, ErrInvalidConfig
	}
	owned := false
	defer func() {
		if !owned {
			_ = p.Close()
		}
	}()

	sourceName := p.Source()
	mc := resolveMetrics(o.metrics)
	tc := resolveTracer(o.tracer)
	lg := resolveLogger(o.logger)
	// The watch outlives the boot load. Keep the caller's values but not the
	// load span, so later reload spans are not children of a finished span.
	baseCtx := ctx
	start := time.Now()
	ctx, span := startLoadSpan(ctx, tc, sourceName)
	defer func() {
		endLoadSpan(span, err)
		recordMetric(ctx, mc, sourceName, start, err, errorClass(err))
	}()

	formatter, ownsFormat := p.(confcore.Formatter)
	leaves, err := derive(tType, ownsFormat)
	if err != nil {
		return nil, err
	}
	if len(leaves) == 0 {
		return nil, fmt.Errorf("%w: %s has no Getter fields", ErrInvalidType, tType)
	}

	keys := make([]string, len(leaves))
	for i, l := range leaves {
		keys[i] = l.key
	}

	raw, rev, err := p.Load(ctx, keys)
	if err != nil {
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
		return nil, errors.Join(missing...)
	}

	values := make(map[string]any, len(leaves))
	for _, l := range leaves {
		v, err := parseLeaf(formatter, raw[l.key], l)
		if err != nil {
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
		source:    sourceName,
		metrics:   o.metrics,
		tracer:    o.tracer,
		logger:    o.logger,
		logCtx:    context.WithoutCancel(baseCtx),
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
			return nil, fmt.Errorf("%w: field %s cannot be bound", ErrInvalidType, l.field)
		}
		binder.bind(func() (any, error) { return core.get(key) })
	}

	data, ok := reflect.TypeAssert[T](dataVal)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected config struct type %s", ErrInvalidType, dataVal.Type())
	}
	loader = &Loader[T]{core: core, data: data}

	if w, ok := p.(confcore.Watcher); ok {
		watchCtx, cancel := context.WithCancel(context.WithoutCancel(baseCtx))
		ch, err := w.Watch(watchCtx, rev)
		if err != nil {
			cancel()
			return nil, err
		}
		core.cancel = cancel
		core.done = make(chan struct{})
		go core.consume(watchCtx, ch)
	}

	lg.Info(ctx, "conf: loaded", "conf.source", sourceName, "conf.revision", rev)
	owned = true
	return loader, nil
}

// Data returns the config struct with every Getter bound. Each call returns a
// copy of the struct; its Getters all read the loader's current snapshot, so a
// copy taken before a reload still returns the latest values.
func (l *Loader[T]) Data() T { return l.data }

// State reports the loader's health and lifecycle state.
func (l *Loader[T]) State() State { return l.core.State() }

// Close stops the watch and closes the provider. It is idempotent. After
// Close, Get returns ErrClosed and State returns StateClosed.
func (l *Loader[T]) Close() error { return l.core.Close() }

// parseLeaf decodes one raw value for l. A Formatter provider owns decoding;
// otherwise the leaf's declared format does.
func parseLeaf(formatter confcore.Formatter, raw string, l leaf) (any, error) {
	if formatter == nil {
		return confcore.ParseValue(raw, l.typ, confcore.Format(l.format))
	}
	v, err := formatter.Parse(raw, l.typ)
	if err != nil {
		return nil, err
	}
	if v == nil {
		if isNilable(l.typ) {
			return reflect.Zero(l.typ).Interface(), nil
		}
		return nil, fmt.Errorf("%w: formatter returned nil for %s", ErrInvalidType, l.typ)
	}
	rv := reflect.ValueOf(v)
	if rv.Type() != l.typ {
		if !rv.Type().AssignableTo(l.typ) {
			return nil, fmt.Errorf("%w: formatter returned %T for %s", ErrInvalidType, v, l.typ)
		}
		// Normalize assignable-but-unnamed results so Get's exact type
		// assertion sees the leaf's declared type.
		v = rv.Convert(l.typ).Interface()
	}
	return v, nil
}

// isNilable reports whether values of t can be nil.
func isNilable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

// State reports the loader's health and lifecycle state.
func (c *loaderCore) State() State {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state
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
			c.loggerClient().Error(c.logCtx, "conf: close failed", "error.type", errorClass(c.closeErr))
		}
	})
	return c.closeErr
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

func (c *loaderCore) setState(s State) {
	c.stateMu.Lock()
	c.state = s
	c.stateMu.Unlock()
}
