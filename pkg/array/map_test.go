package array

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	t.Run("maps int slice to string slice", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Map(input, func(n int) string {
			return fmt.Sprintf("num_%d", n)
		})

		expected := []string{"num_1", "num_2", "num_3", "num_4", "num_5"}
		require.Equal(t, expected, result)
		require.Len(t, result, len(input))
	})

	t.Run("maps string slice to int slice", func(t *testing.T) {
		input := []string{"10", "20", "30", "40"}
		result := Map(input, func(s string) int {
			n, err := strconv.Atoi(s)
			require.NoError(t, err)
			return n
		})

		expected := []int{10, 20, 30, 40}
		require.Equal(t, expected, result)
		require.Len(t, result, len(input))
	})

	t.Run("handles empty slice", func(t *testing.T) {
		input := []int{}
		callCount := 0
		result := Map(input, func(n int) string {
			callCount++
			return "should not be called"
		})

		require.Empty(t, result)
		require.Equal(t, 0, callCount, "transformation function should not be called for empty slice")
	})

	t.Run("works with single element", func(t *testing.T) {
		input := []int{42}
		result := Map(input, func(n int) string {
			return fmt.Sprintf("value_%d", n)
		})

		expected := []string{"value_42"}
		require.Equal(t, expected, result)
	})

	t.Run("works with complex struct transformation", func(t *testing.T) {
		type Person struct {
			ID   int
			Name string
			Age  int
		}

		type PersonSummary struct {
			DisplayName string
			IsAdult     bool
		}

		people := []Person{
			{ID: 1, Name: "Alice", Age: 25},
			{ID: 2, Name: "Bob", Age: 17},
			{ID: 3, Name: "Charlie", Age: 35},
		}

		result := Map(people, func(p Person) PersonSummary {
			return PersonSummary{
				DisplayName: fmt.Sprintf("%s (%d)", p.Name, p.ID),
				IsAdult:     p.Age >= 18,
			}
		})

		expected := []PersonSummary{
			{DisplayName: "Alice (1)", IsAdult: true},
			{DisplayName: "Bob (2)", IsAdult: false},
			{DisplayName: "Charlie (3)", IsAdult: true},
		}
		require.Equal(t, expected, result)
	})

	t.Run("transformation function is called exactly once per element", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		callCount := 0
		processedValues := make([]int, 0)

		result := Map(input, func(n int) int {
			callCount++
			processedValues = append(processedValues, n)
			return n * 2
		})

		require.Equal(t, len(input), callCount, "function should be called exactly once per element")
		require.Equal(t, input, processedValues, "function should be called with each element exactly once")
		require.Equal(t, []int{2, 4, 6, 8, 10}, result)
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		original := []int{1, 2, 3, 4, 5}
		originalCopy := make([]int, len(original))
		copy(originalCopy, original)

		Map(original, func(n int) int { return n * 10 })

		require.Equal(t, originalCopy, original, "original slice should not be modified")
	})

	t.Run("creates independent result slice", func(t *testing.T) {
		input := []int{1, 2, 3}
		result1 := Map(input, func(n int) int { return n * 2 })
		result2 := Map(input, func(n int) int { return n * 3 })

		// Results should be different
		require.NotEqual(t, result1, result2)
		require.Equal(t, []int{2, 4, 6}, result1)
		require.Equal(t, []int{3, 6, 9}, result2)

		// Modifying one result should not affect the other
		result1[0] = 999
		require.Equal(t, []int{999, 4, 6}, result1)
		require.Equal(t, []int{3, 6, 9}, result2)
	})

	t.Run("works with different types", func(t *testing.T) {
		// Bool to string
		bools := []bool{true, false, true}
		boolStrings := Map(bools, func(b bool) string {
			if b {
				return "yes"
			}
			return "no"
		})
		require.Equal(t, []string{"yes", "no", "yes"}, boolStrings)

		// Float to int (truncation)
		floats := []float64{1.1, 2.7, 3.9, 4.2}
		ints := Map(floats, func(f float64) int { return int(f) })
		require.Equal(t, []int{1, 2, 3, 4}, ints)
	})

	t.Run("handles slice extraction from struct", func(t *testing.T) {
		type User struct {
			ID    int
			Name  string
			Email string
		}

		users := []User{
			{ID: 1, Name: "Alice", Email: "alice@example.com"},
			{ID: 2, Name: "Bob", Email: "bob@example.com"},
			{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
		}

		// Extract IDs
		ids := Map(users, func(u User) int { return u.ID })
		require.Equal(t, []int{1, 2, 3}, ids)

		// Extract names
		names := Map(users, func(u User) string { return u.Name })
		require.Equal(t, []string{"Alice", "Bob", "Charlie"}, names)

		// Extract emails
		emails := Map(users, func(u User) string { return u.Email })
		require.Equal(t, []string{"alice@example.com", "bob@example.com", "charlie@example.com"}, emails)
	})

	t.Run("preserves order", func(t *testing.T) {
		input := []int{5, 3, 8, 1, 9, 2, 7, 4, 6}
		result := Map(input, func(n int) string { return fmt.Sprintf("%d", n) })

		// Result should maintain the same order as input
		expected := []string{"5", "3", "8", "1", "9", "2", "7", "4", "6"}
		require.Equal(t, expected, result)
	})

	t.Run("handles large slices efficiently", func(t *testing.T) {
		const largeSize = 100000
		input := make([]int, largeSize)
		for i := range input {
			input[i] = i
		}

		result := Map(input, func(n int) int { return n * 2 })

		require.Len(t, result, largeSize)
		require.Equal(t, largeSize, cap(result), "capacity should equal length for efficiency")

		// Verify a few elements
		require.Equal(t, 0, result[0])
		require.Equal(t, 2, result[1])
		require.Equal(t, (largeSize-1)*2, result[largeSize-1])
	})

	t.Run("works with pointer types", func(t *testing.T) {
		values := []*int{intPtr(10), intPtr(20), intPtr(30)}
		result := Map(values, func(ptr *int) int {
			if ptr == nil {
				return 0
			}
			return *ptr
		})

		expected := []int{10, 20, 30}
		require.Equal(t, expected, result)
	})

	t.Run("handles nil pointers in transformation", func(t *testing.T) {
		values := []*int{intPtr(10), nil, intPtr(30)}
		result := Map(values, func(ptr *int) int {
			if ptr == nil {
				return -1
			}
			return *ptr
		})

		expected := []int{10, -1, 30}
		require.Equal(t, expected, result)
	})
}

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}

// Example tests for documentation
func ExampleMap() {
	numbers := []int{1, 2, 3, 4, 5}
	strings := Map(numbers, func(n int) string {
		return fmt.Sprintf("num_%d", n)
	})

	fmt.Printf("Mapped %d numbers to strings\n", len(strings))
	fmt.Printf("First result: %s\n", strings[0])
	// Output:
	// Mapped 5 numbers to strings
	// First result: num_1
}

func ExampleMap_structTransformation() {
	type User struct {
		ID   int
		Name string
	}

	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	userIDs := Map(users, func(u User) int { return u.ID })

	fmt.Printf("Extracted IDs: %v\n", userIDs)
	// Output: Extracted IDs: [1 2]
}

func ExampleMap_typeConversion() {
	strings := []string{"10", "20", "30"}
	numbers := Map(strings, func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}
		return n
	})

	fmt.Printf("Converted to numbers: %v\n", numbers)
	// Output: Converted to numbers: [10 20 30]
}

// Benchmark tests for Map function
func BenchmarkMap(b *testing.B) {
	b.Run("Map_1000_int_to_string", func(b *testing.B) {
		input := make([]int, 1000)
		for i := range input {
			input[i] = i
		}
		b.ResetTimer()

		for b.Loop() {
			Map(input, func(n int) string { return strconv.Itoa(n) })
		}
	})

	b.Run("Map_10000_struct_field_extraction", func(b *testing.B) {
		type Item struct{ Value int }
		input := make([]Item, 10000)
		for i := range input {
			input[i] = Item{Value: i}
		}
		b.ResetTimer()

		for b.Loop() {
			Map(input, func(item Item) int { return item.Value })
		}
	})

	b.Run("Map_100000_simple_calculation", func(b *testing.B) {
		input := make([]int, 100000)
		for i := range input {
			input[i] = i
		}
		b.ResetTimer()

		for b.Loop() {
			Map(input, func(n int) int { return n * 2 })
		}
	})
}

// Test performance comparison with manual approaches
func BenchmarkMapVsManual(b *testing.B) {
	const size = 10000
	input := make([]int, size)
	for i := range input {
		input[i] = i
	}

	b.Run("Map", func(b *testing.B) {
		for b.Loop() {
			Map(input, func(n int) string { return strconv.Itoa(n) })
		}
	})

	b.Run("Manual_make_and_loop", func(b *testing.B) {
		for b.Loop() {
			result := make([]string, len(input))
			for j, v := range input {
				result[j] = strconv.Itoa(v)
			}
		}
	})

	b.Run("Append_based", func(b *testing.B) {
		for b.Loop() {
			var result []string
			for _, v := range input {
				result = append(result, strconv.Itoa(v))
			}
			_ = result
		}
	})
}
