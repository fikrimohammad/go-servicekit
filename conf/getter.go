package conf

import (
	"fmt"
	"reflect"
)

// Getter is a typed accessor for one config key.
type Getter[T any] struct {
	// Core returns the current raw value. It is set by the loader and is
	// opaque to callers. Use Get or GetWithDefault.
	Core func() (any, error)
}

// Get returns the latest value. It returns ErrUnbound for a zero Getter,
// the underlying error when the value is absent or the loader is closed, and
// the zero value with an error when the stored value has the wrong type.
func (g Getter[T]) Get() (T, error) {
	var zero T
	if g.Core == nil {
		return zero, ErrUnbound
	}
	v, err := g.Core()
	if err != nil {
		return zero, err
	}
	tv, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("%w: expected %T, got %T", ErrInvalidType, zero, v)
	}
	return tv, nil
}

// GetWithDefault returns the latest value, or def when the value is absent,
// the Getter is unbound, the loader is closed, or the stored value has the
// wrong type. It never fails.
func (g Getter[T]) GetWithDefault(def T) T { // gostyle:ignore getter-prefix GetWithDefault pairs with Get on Getter; the concept is literally a get.
	if g.Core == nil {
		return def
	}
	v, err := g.Core()
	if err != nil {
		return def
	}
	tv, ok := v.(T)
	if !ok {
		return def
	}
	return tv
}

// bind attaches the loader's raw accessor to this Getter.
func (g *Getter[T]) bind(core func() (any, error)) {
	g.Core = core
}

// target returns the value type T.
func (g *Getter[T]) target() reflect.Type { return reflect.TypeFor[T]() }
