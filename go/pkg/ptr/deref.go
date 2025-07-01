package ptr

// SafeDeref returns the value pointed to by p or a fallback value if p is nil.
// When p is nil and no fallback is provided, it returns the zero value for type T.
//
// The function accepts any type T through its generic parameter. The fallback
// parameter is variadic for convenience, but only the first value is used if
// multiple are provided.
//
// SafeDeref is particularly useful when working with optional values from APIs,
// database results, or configurations, where nil checks would otherwise clutter
// the code. It helps avoid nil pointer panics without sacrificing readability.
//
// For types with reference semantics (maps, slices), the zero value is nil, not
// an initialized empty container.
//
// Example usage:
//
//	s := "hello"
//	fmt.Println(ptr.SafeDeref(&s))        // Prints: hello
//
//	var nilStr *string
//	fmt.Println(ptr.SafeDeref(nilStr))    // Prints: "" (zero value)
//	fmt.Println(ptr.SafeDeref(nilStr, "default")) // Prints: default
//
//	// In a function handling a struct with pointer fields
//	func processConfig(cfg *Config) {
//	    timeout := ptr.SafeDeref(cfg.Timeout, 30)
//	    retries := ptr.SafeDeref(cfg.Retries, 3)
//	    // Continue with non-nil values
//	}
//
// SafeDeref is safe for concurrent use as it performs no mutation of shared state
// and makes no allocations beyond those needed for the return value.
func SafeDeref[T any](p *T, fallback ...T) T {
	if p == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		var safe T
		return safe
	}
	return *p
}
