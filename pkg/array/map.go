package array

// Map creates a new slice by applying the transformation function to each element of the input slice.
//
// The transformation function is called exactly once per element in sequential order.
// The result slice has the same length as the input slice.
//
// Parameters:
//
//   - arr: Input slice to transform
//
//   - fn: Function applied to each element to produce the transformed value
//
//     // Extract fields from structs
//     users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
//     userIDs := array.Map(users, func(u User) int { return u.ID })
//     // Result: []int{1, 2}
//
//     // Convert between types
//     numbers := []int{1, 2, 3}
//     strings := array.Map(numbers, func(n int) string { return strconv.Itoa(n) })
//     // Result: []string{"1", "2", "3"}
func Map[T any, R any](arr []T, fn func(T) R) []R {
	result := make([]R, len(arr))
	for i := range arr {
		result[i] = fn(arr[i])
	}
	return result
}
