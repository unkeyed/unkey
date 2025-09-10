package array

// Reduce aggregates all elements of a slice into a single value using the reduction function.
//
// Processes each element in order, combining it with the accumulator using the provided function.
// The reduction function receives the current accumulator and the current element.
//
// Parameters:
//
//   - arr: Input slice to reduce
//
//   - fn: Function that combines the accumulator with each element
//
//   - init: Initial accumulator value, returned if slice is empty
//
//     // Sum numbers
//     numbers := []int{1, 2, 3, 4, 5}
//     sum := array.Reduce(numbers, func(acc, val int) int { return acc + val }, 0)
//     // Result: 15
//
//     // Build frequency map
//     words := []string{"hello", "world", "hello"}
//     freq := array.Reduce(words, func(acc map[string]int, word string) map[string]int {
//     acc[word]++
//     return acc
//     }, make(map[string]int))
//     // Result: map[string]int{"hello": 2, "world": 1}
func Reduce[T any, R any](arr []T, fn func(R, T) R, init R) R {
	result := init
	for _, v := range arr {
		result = fn(result, v)
	}
	return result
}
