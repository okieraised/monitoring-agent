package utilities

import "reflect"

// Ptr returns a pointer to v.
func Ptr[T any](v T) *T { return &v }

// Deref returns the value pointed to by p, or def if p is nil.
func Deref[T any](p *T) T {
	return *p
}

// PtrOrNil returns a pointer to v, or nil if v is the zero value.
func PtrOrNil[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// DerefZero returns the value pointed to by p, or the zero value if p is nil.
func DerefZero[T any](p *T) T {
	if p != nil {
		return *p
	}
	var zero T
	return zero
}

// IsNil reports whether p is nil.
func IsNil[T any](p *T) bool { return p == nil }

// IsZero reports whether p is non-nil and points to a zero value.
func IsZero[T comparable](p *T) bool {
	if p == nil {
		return false
	}
	return *p == *new(T)
}

// IsZeroOrNil reports whether p is nil or points to a zero value.
func IsZeroOrNil[T comparable](p *T) bool {
	return p == nil || *p == *new(T)
}

// Equal reports whether both pointers are nil or point to equal values.
func Equal[T comparable](a, b *T) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

// Same reports whether two pointers refer to the same underlying value.
func Same[T any](a, b *T) bool { return a == b }

// Clone returns a pointer to a copy of the value pointed to by p, or nil if p is nil.
func Clone[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// CopyToPtr creates a new pointer with a copy of another pointer's value, or to def if nil.
func CopyToPtr[T any](p *T, def T) *T {
	if p != nil {
		v := *p
		return &v
	}
	return &def
}

// SetIfNil sets *p to a pointer to v if *p is nil.
func SetIfNil[T any](p **T, v T) {
	if *p == nil {
		*p = &v
	}
}

// ZeroIfNil returns a non-nil pointer, using zero value if p is nil.
func ZeroIfNil[T any](p *T) *T {
	if p == nil {
		var zero T
		return &zero
	}
	return p
}

// SliceOfPtrs returns a slice of pointers to each element of vals.
func SliceOfPtrs[T any](vals []T) []*T {
	out := make([]*T, len(vals))
	for i := range vals {
		out[i] = &vals[i]
	}
	return out
}

// PtrsToValues dereferences each pointer in ptrs into a slice of values.
// Nil pointers become zero values.
func PtrsToValues[T any](ptrs []*T) []T {
	out := make([]T, len(ptrs))
	for i, p := range ptrs {
		if p != nil {
			out[i] = *p
		}
	}
	return out
}

// IsNilInterface reports whether an interface is nil or contains a nil pointer.
func IsNilInterface(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
