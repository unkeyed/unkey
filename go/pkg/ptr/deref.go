package ptr

// SafeDeref safely dereferences a pointer, returning a zero value if the pointer is nil.
//
// SafeDeref is useful in scenarios where you need to dereference a pointer but want to avoid
// potential nil pointer dereference errors. This function ensures that a zero value of the
// specified type is returned if the pointer is nil, providing a safe fallback.
//
// Parameters:
// - p: A pointer to a value of any type.
//
// Returns:
// - The value pointed to by p if p is not nil; otherwise, the zero value of the type T.
//
// Example Usage:
// ```go
// package main
//
// import (
//
//	"fmt"
//	"unkey/go/pkg/ptr"
//
// )
//
//	func main() {
//	    var p *int
//	    value := ptr.SafeDeref(p)
//	    fmt.Println(value) // Output: 0
//
//	    x := 42
//	    p = &x
//	    value = ptr.SafeDeref(p)
//	    fmt.Println(value) // Output: 42
//	}
//
// ```
//
// Non-obvious behaviors, edge cases, and limitations:
// - SafeDeref will always return the zero value for the type T if the pointer is nil.
// - This function does not handle nested pointers or pointers to pointers.
//
// Anti-patterns:
//   - Do not use SafeDeref to check for nil pointers in performance-critical code where the
//     overhead of the function call might be significant. Instead, perform a direct nil check.
func SafeDeref[T any](p *T) T {
	if p == nil {
		var safe T
		return safe
	}
	return *p
}
