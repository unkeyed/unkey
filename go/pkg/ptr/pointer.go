package ptr

// P returns a pointer to a new copy of the value v.
//
// This is a generic function that works with any type. It creates a copy
// of the provided value and returns a pointer to this copy. The value is
// stored in the heap, not the stack.
//
// Parameters:
//   - t: The value to create a pointer to. Can be of any type, including
//     primitive types, structs, or interfaces.
//
// Returns:
//   - *T: A pointer to a new copy of the value t.
//
// Thread Safety:
//
//	This function is thread-safe as it doesn't access or modify any shared state.
//
// Performance Considerations:
//
//	Creating a pointer with P() allocates memory on the heap. For performance-critical
//	code paths where many pointers are created in tight loops, consider reusing
//	a single variable and taking its address instead.
//
// Examples:
//
//	Basic usage with a string:
//	   str := "hello"
//	   strPtr := ptr.P(str)
//	   // Now strPtr is *string pointing to a copy of "hello"
//
//	With structs:
//	   type User struct {
//	       Name *string
//	       Age  *int
//	   }
//
//	   user := User{
//	       Name: ptr.P("Alice"),
//	       Age:  ptr.P(30),
//	   }
//
//	Common pattern for API parameters:
//	   client.UpdateConfig(&api.Config{
//	       Timeout: ptr.P(60),
//	       Retries: ptr.P(3),
//	   })
func P[T any](t T) *T {
	return &t
}
