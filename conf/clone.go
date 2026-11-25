package conf

import "reflect"

// cloneVisit identifies an already cloned pointer, map, or slice so cycles and
// shared references keep their relationship in the copy.
type cloneVisit struct {
	typ reflect.Type
	ptr uintptr
	len int
}

// hasReference reports whether values of t expose mutable reference data that
// cloneValue can set. Unexported struct fields are shallow-copied, so they do
// not force a deep copy.
func hasReference(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Interface:
		return true
	case reflect.Array:
		return hasReference(t.Elem())
	case reflect.Struct:
		for f := range t.Fields() {
			if f.IsExported() && hasReference(f.Type) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// cloneGetterValue returns a deep copy of v. Exported struct fields are cloned;
// unexported fields are copied shallowly because reflection cannot set them.
func cloneGetterValue[T any](v T) T {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return v
	}
	return cloneValue(rv, make(map[cloneVisit]reflect.Value)).Interface().(T)
}

// cloneValue returns a deep copy of v. seen preserves shared references and
// stops cycles.
func cloneValue(v reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(cloneValue(v.Elem(), seen))
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		key := cloneVisit{typ: v.Type(), ptr: v.Pointer()}
		if out, ok := seen[key]; ok {
			return out
		}
		elem := reflect.New(v.Type().Elem())
		out := elem
		if out.Type() != v.Type() {
			// Preserve a defined pointer type such as type Optional *int.
			out = out.Convert(v.Type())
		}
		seen[key] = out
		elem.Elem().Set(cloneValue(v.Elem(), seen))
		return out
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		key := cloneVisit{typ: v.Type(), ptr: v.Pointer()}
		if out, ok := seen[key]; ok {
			return out
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		seen[key] = out
		for iter := v.MapRange(); iter.Next(); {
			out.SetMapIndex(cloneValue(iter.Key(), seen), cloneValue(iter.Value(), seen))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		if v.Len() == 0 {
			return reflect.MakeSlice(v.Type(), 0, 0)
		}
		key := cloneVisit{typ: v.Type(), ptr: v.Pointer(), len: v.Len()}
		if out, ok := seen[key]; ok {
			return out
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		seen[key] = out
		if !hasReference(v.Type().Elem()) {
			reflect.Copy(out, v)
			return out
		}
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(cloneValue(v.Index(i), seen))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(cloneValue(v.Index(i), seen))
		}
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		for i := 0; i < v.NumField(); i++ {
			if !out.Field(i).CanSet() {
				continue
			}
			out.Field(i).Set(cloneValue(v.Field(i), seen))
		}
		return out
	default:
		return v
	}
}
