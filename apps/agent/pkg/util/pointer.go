package util

// Pointer returns a pointer of the value passed in.
//
// Because you pass in an argument by value, a copy is made.
// modfying the pointer will not modify the original value.
//
// See the tests for this function for examples.
func Pointer[T any](t T) *T {
	return &t
}
