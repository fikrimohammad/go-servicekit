package conf

import (
	"fmt"
	"reflect"
)

// Getter is a typed accessor for one config key.
//
// A Loader creates and binds Getters; the type has no exported fields, and its
// zero value is unbound. [Getter.Get] returns [ErrUnbound] for a zero Getter,
// and [Getter.GetWithDefault] returns its default. A Getter is safe to copy and
// to read concurrently: every copy observes the loader's latest snapshot.
//
// Reference values such as maps, slices, and pointers are deep-copied on every
// read, so callers can mutate the result without changing the snapshot or
// racing with other readers. Mutable state hidden behind unexported fields is
// copied shallowly.
type Getter[T any] struct {
	core    func() (any, error)
	deep    bool
	dynamic bool
}

// Get returns a copy of the latest value. It returns ErrUnbound for a zero
// Getter, the underlying error when the value is absent or the loader is
// closed, and the zero value with an error when the stored value has the wrong
// type.
//
// Maps, slices, pointers, arrays, and exported struct fields are deep-copied.
// Mutating the returned value does not change the loader's snapshot.
func (g Getter[T]) Get() (T, error) {
	var zero T
	if g.core == nil {
		return zero, ErrUnbound
	}
	v, err := g.core()
	if err != nil {
		return zero, err
	}
	tv, ok := assertValue[T](v)
	if !ok {
		return zero, fmt.Errorf("%w: expected %T, got %T", ErrInvalidType, zero, v)
	}
	return g.value(tv), nil
}

// GetWithDefault returns a copy of the latest value, or def when the value is
// absent, the Getter is unbound, the loader is closed, or the stored value has
// the wrong type. It never fails.
func (g Getter[T]) GetWithDefault(def T) T { // gostyle:ignore getter-prefix GetWithDefault pairs with Get on Getter; the concept is literally a get.
	if g.core == nil {
		return def
	}
	v, err := g.core()
	if err != nil {
		return def
	}
	tv, ok := assertValue[T](v)
	if !ok {
		return def
	}
	return g.value(tv)
}

// assertValue converts v to T. A nil value succeeds when T can be nil.
func assertValue[T any](v any) (T, bool) {
	var zero T
	if v == nil {
		return zero, isNilable(reflect.TypeFor[T]())
	}
	tv, ok := v.(T)
	return tv, ok
}

// value returns tv, deep-copied when it exposes mutable reference data.
func (g Getter[T]) value(tv T) T {
	if !g.deep {
		return tv
	}
	if g.dynamic {
		t := reflect.TypeOf(tv)
		if t == nil || !hasReference(t) {
			return tv
		}
	}
	return cloneGetterValue(tv)
}

// bind attaches the loader's raw accessor to this Getter.
func (g *Getter[T]) bind(core func() (any, error)) {
	g.core = core
	t := reflect.TypeFor[T]()
	g.deep = hasReference(t)
	g.dynamic = t.Kind() == reflect.Interface
}

// target returns the value type T.
func (g *Getter[T]) target() reflect.Type { return reflect.TypeFor[T]() }

var _ getterLeaf = (*Getter[any])(nil)
