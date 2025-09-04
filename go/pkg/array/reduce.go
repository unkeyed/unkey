package array

// Reduce aggregates all elements of a slice into a single value using the provided reduction function.
//
// This function implements the classic functional programming reduce operation (also known as fold),
// which processes each element of the slice in order, accumulating a result value according to the
// provided reduction function. The operation starts with an initial accumulator value and applies
// the reduction function to combine each element with the current accumulator.
//
// Reduce is designed for efficient aggregation operations such as summing values, building maps or
// sets from slices, concatenating strings, finding maximum/minimum values, or performing complex
// data transformations that require maintaining state across iterations. The sequential processing
// ensures predictable behavior and allows for operations that depend on order.
//
// Parameters:
//   - arr: The input slice to reduce. Can be empty, in which case the initial value is returned
//     unchanged. The input slice is never modified by the operation. Type T can be any Go type
//     including primitives, structs, interfaces, or other slices.
//   - fn: The reduction function that combines the current accumulator with each element. Called
//     exactly once per element in sequential order from index 0 to len(arr)-1. The function
//     receives the current accumulator value as the first parameter and the current element as
//     the second parameter. Must return the new accumulator value. If the reduction function
//     panics, Reduce propagates the panic. The function should be associative for predictable
//     results, though this is not enforced.
//   - init: The initial accumulator value used as the starting point for the reduction. This value
//     is passed to the first invocation of the reduction function, or returned directly if the
//     slice is empty. The type R can be different from the element type T, allowing for
//     transformative reductions.
//
// Returns the final accumulator value after processing all elements. For empty slices, returns
// the initial value unchanged. The result type R can be the same as or different from the input
// element type T, enabling type-transforming reductions.
//
// Behavior details:
//   - Reduction function is called exactly len(arr) times in sequential order
//   - Input slice is never modified, even temporarily
//   - Each function call receives the result of the previous call as the accumulator
//   - Empty input slice returns init immediately, reduction function never called
//   - Order of processing is guaranteed to be from index 0 to len(arr)-1
//
// Performance characteristics:
//   - Time complexity: O(n) where n equals len(arr)
//   - Space complexity: O(1) constant space for the reduction operation itself
//   - Memory allocation: No memory allocation for the reduction operation (though the reduction
//     function may allocate memory)
//   - Function calls: Exactly len(arr) calls to the reduction function
//
// The sequential, single-pass nature of Reduce makes it highly efficient for aggregation
// operations. Performance is directly proportional to the cost of the reduction function
// and any memory allocations it performs.
//
// Concurrency considerations:
//
//	Reduce is safe to call concurrently from multiple goroutines when operating on different
//	input slices. Concurrent Reduce operations on the same input slice are safe since the input
//	is never modified. The safety of concurrent Reduce operations depends on the reduction
//	function and the accumulator type - if the function or accumulator involves shared mutable
//	state without synchronization, race conditions may occur.
//
// Error conditions:
//
//	Reduce does not return errors as slice iteration cannot fail under normal conditions.
//	Reduction function panics are propagated to the caller unchanged. The accumulator value
//	is not validated - it is the caller's responsibility to ensure the initial value and
//	reduction function are compatible.
//
// Edge cases and special behavior:
//   - Empty input slice: Returns init immediately, reduction function never called
//   - Single element slice: Reduction function called once with (init, element)
//   - Nil reduction function: Will panic when called, following Go's standard behavior
//   - Large slices: Performance scales linearly with slice length
//
// Common usage patterns:
//
//	// Sum numeric values
//	numbers := []int{1, 2, 3, 4, 5}
//	sum := array.Reduce(numbers, func(acc, val int) int { return acc + val }, 0)
//	// Result: 15
//
//	// Find maximum value
//	values := []float64{3.14, 2.71, 1.41, 4.67, 2.23}
//	max := array.Reduce(values, func(acc, val float64) float64 {
//	    if val > acc { return val }
//	    return acc
//	}, math.Inf(-1))
//	// Result: 4.67
//
//	// Build frequency map
//	words := []string{"hello", "world", "hello", "go", "world", "hello"}
//	freq := array.Reduce(words, func(acc map[string]int, word string) map[string]int {
//	    acc[word]++
//	    return acc
//	}, make(map[string]int))
//	// Result: map[string]int{"hello": 3, "world": 2, "go": 1}
//
//	// Concatenate with separator
//	items := []string{"apple", "banana", "cherry"}
//	joined := array.Reduce(items, func(acc, item string) string {
//	    if acc == "" { return item }
//	    return acc + ", " + item
//	}, "")
//	// Result: "apple, banana, cherry"
//
//	// Transform slice to different type
//	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
//	nameMap := array.Reduce(users, func(acc map[int]string, user User) map[int]string {
//	    acc[user.ID] = user.Name
//	    return acc
//	}, make(map[int]string))
//	// Result: map[int]string{1: "Alice", 2: "Bob"}
//
// Integration with other array functions:
//
//	Reduce works seamlessly with other array package functions, particularly [Map] for
//	map-reduce operations. It can process data generated by [Fill] or work with elements
//	selected by [Random]:
//
//	// Map-reduce pipeline: transform then aggregate
//	scores := []int{85, 92, 78, 96, 88}
//	letterGrades := array.Map(scores, func(score int) string {
//	    if score >= 90 { return "A" }
//	    if score >= 80 { return "B" }
//	    return "C"
//	})
//	gradeCount := array.Reduce(letterGrades, func(acc map[string]int, grade string) map[string]int {
//	    acc[grade]++
//	    return acc
//	}, make(map[string]int))
//
//	// Aggregate generated test data
//	testResults := array.Fill(10000, func() TestResult { return generateTestResult() })
//	successRate := array.Reduce(testResults, func(acc float64, result TestResult) float64 {
//	    if result.Success { return acc + 1.0 }
//	    return acc
//	}, 0.0) / float64(len(testResults))
//
// Mathematical properties and associativity:
//
//	While Reduce processes elements sequentially, many reduction operations are associative,
//	meaning the order of combination doesn't affect the final result for mathematical operations:
//
//	// Associative operations (order-independent results):
//	sum := array.Reduce(numbers, func(a, b int) int { return a + b }, 0)        // Addition
//	product := array.Reduce(numbers, func(a, b int) int { return a * b }, 1)   // Multiplication
//	max := array.Reduce(numbers, func(a, b int) int { max(a, b) }, math.MinInt) // Maximum
//
//	// Non-associative operations (order-dependent results):
//	diff := array.Reduce(numbers, func(a, b int) int { return a - b }, 0)      // Subtraction
//	concat := array.Reduce(strings, func(a, b string) string { return a + b }, "") // String concatenation
//
// Performance comparison with alternatives:
//
//	// Reduce - optimal for single-pass aggregation
//	result := array.Reduce(data, aggregateFunc, initialValue)
//
//	// Manual loop - equivalent performance but more verbose
//	result := initialValue
//	for _, item := range data {
//	    result = aggregateFunc(result, item)
//	}
//
//	// Multiple passes - less efficient for complex aggregations
//	count := len(data)
//	sum := 0
//	for _, item := range data { sum += item.Value }      // First pass
//	max := math.MinInt
//	for _, item := range data { max = max(max, item.Value) } // Second pass
//	// Reduce can do both in a single pass
//
// Anti-patterns to avoid:
//
//	// Bad: Using Reduce for operations that don't need accumulation
//	array.Reduce(users, func(acc []string, user User) []string {
//	    fmt.Println(user.Name)  // Side effect
//	    return append(acc, user.Name)
//	}, []string{})
//	// Good: Use Map for transformations, simple loop for side effects
//	names := array.Map(users, func(u User) string { return u.Name })
//	for _, name := range names { fmt.Println(name) }
//
//	// Bad: Building large intermediate collections unnecessarily
//	result := array.Reduce(largeData, func(acc []ProcessedItem, item Item) []ProcessedItem {
//	    return append(acc, processExpensively(item))  // Grows slice repeatedly
//	}, []ProcessedItem{})
//	// Good: Pre-allocate or use different approach for large transformations
//	result := array.Map(largeData, processExpensively)
//
//	// Bad: Non-associative operations where order matters but isn't considered
//	// This may produce unexpected results if mental model assumes commutativity
//	difference := array.Reduce(numbers, func(a, b int) int { return a - b }, 0)
//	// Good: Be explicit about order-dependent operations
//	leftToRight := array.Reduce(numbers, func(acc, val int) int { return acc - val }, 0)
func Reduce[T any, R any](arr []T, fn func(R, T) R, init R) R {
	result := init
	for _, v := range arr {
		result = fn(result, v)
	}
	return result
}
