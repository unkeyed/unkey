package array

// Map creates a new slice by applying the provided transformation function to each element of the input slice.
//
// This function implements the classic functional programming map operation, transforming each element
// from type T to type R according to the provided function. The transformation preserves order and
// length - the result slice will have exactly the same length as the input slice, with each element
// at index i being the result of applying the transformation function to the input element at index i.
//
// Map is designed for efficient bulk transformations of slices, particularly useful for converting
// between different data types, extracting fields from structs, or applying calculations to numeric
// data. The single allocation approach ensures optimal memory usage and performance characteristics
// comparable to manual iteration.
//
// Parameters:
//   - arr: The input slice to transform. Can be empty, in which case an empty result slice is returned.
//     The input slice is never modified by the operation. Type T can be any Go type including primitives,
//     structs, interfaces, or other slices.
//   - fn: The transformation function applied to each element. Called exactly once per input element
//     in sequential order from index 0 to len(arr)-1. The function should be lightweight and preferably
//     side-effect free. If the transformation function panics, Map propagates the panic. Functions that
//     perform I/O or other blocking operations will impact performance proportionally to slice length.
//
// Returns a new slice of type []R containing the transformed elements. The result slice has length
// equal to the input slice length and capacity equal to length, ensuring no additional allocations
// are needed if the slice is not grown further. Each element at index i contains the result of
// calling fn() on the input element at index i.
//
// Behavior details:
//   - Transformation function is called exactly len(arr) times in sequential order
//   - Input slice is never modified, even temporarily
//   - Result slice is completely independent of the input slice
//   - Empty input slice returns empty result slice, transformation function is never called
//   - Slice allocation occurs once at the beginning with capacity = len(arr)
//
// Performance characteristics:
//   - Time complexity: O(n) where n equals len(arr)
//   - Space complexity: O(n) for result slice data plus constant overhead
//   - Memory allocation: Single allocation for result slice, no additional allocations during transformation
//   - Function calls: Exactly len(arr) calls to the transformation function
//
// The single allocation approach makes Map more efficient than append-based transformations for
// known slice sizes, eliminating the exponential growth and copying overhead of slice expansion.
// Performance is directly proportional to the cost of the transformation function.
//
// Concurrency considerations:
//
//	Map is safe to call concurrently from multiple goroutines when operating on different input
//	slices. Concurrent Map operations on the same input slice are safe since the input is never
//	modified. The safety of concurrent Map operations depends on the transformation function - if
//	the function accesses shared mutable state without synchronization, race conditions may occur.
//
// Error conditions:
//
//	Map does not return errors as slice creation and iteration cannot fail under normal conditions.
//	Memory exhaustion during allocation will result in a runtime panic, consistent with Go's make()
//	function. Transformation function panics are propagated to the caller unchanged.
//
// Edge cases and special behavior:
//   - Empty input slice: Returns empty result slice, transformation function never called
//   - Very large input slices: May cause out-of-memory panics if insufficient RAM available
//   - Transformation function panic: Panic propagates to caller, partial result slice may be allocated but not returned
//   - Nil function: Will panic when called, following Go's standard behavior for nil function calls
//
// Common usage patterns:
//
//	// Extract field from struct slice
//	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
//	userIDs := array.Map(users, func(u User) int { return u.ID })
//	// Result: []int{1, 2}
//
//	// Convert types
//	numbers := []int{1, 2, 3, 4, 5}
//	strings := array.Map(numbers, func(n int) string { return fmt.Sprintf("num_%d", n) })
//	// Result: []string{"num_1", "num_2", "num_3", "num_4", "num_5"}
//
//	// Apply calculations
//	prices := []float64{10.0, 20.0, 30.0}
//	withTax := array.Map(prices, func(p float64) float64 { return p * 1.08 })
//	// Result: []float64{10.8, 21.6, 32.4}
//
//	// Transform nested data
//	configs := []Config{{Port: 8080}, {Port: 8081}, {Port: 8082}}
//	endpoints := array.Map(configs, func(c Config) string {
//	    return fmt.Sprintf("localhost:%d", c.Port)
//	})
//	// Result: []string{"localhost:8080", "localhost:8081", "localhost:8082"}
//
// Integration with other array functions:
//
//	Map works seamlessly with other array package functions. It can be combined with [Fill]
//	for data generation pipelines, with [Random] for selecting transformation parameters, or
//	chained with [Reduce] for map-reduce operations:
//
//	// Generate test data and extract specific fields
//	testData := array.Fill(1000, func() TestRecord { return generateTestRecord() })
//	testIDs := array.Map(testData, func(r TestRecord) string { return r.ID })
//
//	// Transform and aggregate in a pipeline
//	scores := []int{85, 92, 78, 96, 88}
//	grades := array.Map(scores, func(score int) string {
//	    if score >= 90 { return "A" }
//	    if score >= 80 { return "B" }
//	    return "C"
//	})
//	gradeCount := array.Reduce(grades, func(acc map[string]int, grade string) map[string]int {
//	    acc[grade]++
//	    return acc
//	}, make(map[string]int))
//
// Performance comparison with alternatives:
//
//	// Map - optimal for transformations
//	result := array.Map(input, transformFunc)
//
//	// Manual loop - equivalent performance but more verbose
//	result := make([]ResultType, len(input))
//	for i, v := range input {
//	    result[i] = transformFunc(v)
//	}
//
//	// Append-based - significantly slower for large slices due to growth overhead
//	var result []ResultType
//	for _, v := range input {
//	    result = append(result, transformFunc(v))
//	}
//
// Anti-patterns to avoid:
//
//	// Bad: Using Map for side effects without using the result
//	array.Map(users, func(u User) User {
//	    fmt.Println(u.Name)  // Side effect, return value ignored
//	    return u
//	})
//	// Good: Use a simple range loop for side effects
//	for _, u := range users {
//	    fmt.Println(u.Name)
//	}
//
//	// Bad: Complex transformation function with multiple responsibilities
//	array.Map(data, func(d Data) Result {
//	    // Log, validate, transform, and format all in one function
//	    log.Printf("Processing %+v", d)
//	    if err := d.Validate(); err != nil { panic(err) }
//	    transformed := d.Process()
//	    return transformed.Format()
//	})
//	// Good: Keep transformation function focused and simple
//	validData := validateData(data)  // Separate validation step
//	result := array.Map(validData, func(d Data) Result { return d.Transform() })
func Map[T any, R any](arr []T, fn func(T) R) []R {
	result := make([]R, len(arr))
	for i, v := range arr {
		result[i] = fn(v)
	}
	return result
}
