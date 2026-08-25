package util

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// Val returns the dereferenced value of a pointer, or fallback if the pointer is nil.
func Val[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// BoolPtr returns a pointer to a boolean value.
func BoolPtr(b bool) *bool {
	return &b
}

// BoolVal returns the value of a boolean pointer or the fallback value if nil.
func BoolVal(b *bool, fallback bool) bool {
	if b == nil {
		return fallback
	}
	return *b
}
