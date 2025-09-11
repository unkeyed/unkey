package array

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFill(t *testing.T) {
	t.Run("creates slice with generated values", func(t *testing.T) {
		counter := 0
		result := Fill(5, func() int {
			counter++
			return counter
		})

		require.Len(t, result, 5)
		require.Equal(t, []int{1, 2, 3, 4, 5}, result)
		require.Equal(t, 5, counter, "generator should be called exactly 5 times")
	})

	t.Run("works with string generation", func(t *testing.T) {
		result := Fill(3, func() string { return "test" })

		require.Len(t, result, 3)
		require.Equal(t, []string{"test", "test", "test"}, result)
	})

	t.Run("returns empty slice for zero length", func(t *testing.T) {
		callCount := 0
		result := Fill(0, func() string {
			callCount++
			return "should not be called"
		})

		require.Empty(t, result)
		require.Equal(t, 0, callCount, "generator should not be called for zero length")
	})

	t.Run("returns empty slice for negative length", func(t *testing.T) {
		callCount := 0
		result := Fill(-5, func() int {
			callCount++
			return 42
		})

		require.Empty(t, result)
		require.Equal(t, 0, callCount, "generator should not be called for negative length")
	})

	t.Run("works with complex types", func(t *testing.T) {
		type User struct {
			ID   int
			Name string
		}

		counter := 0
		result := Fill(2, func() User {
			counter++
			return User{ID: counter, Name: fmt.Sprintf("user%d", counter)}
		})

		expected := []User{
			{ID: 1, Name: "user1"},
			{ID: 2, Name: "user2"},
		}
		require.Len(t, result, 2)
		require.Equal(t, expected, result)
	})

	t.Run("generator function is called exactly once per element", func(t *testing.T) {
		const length = 100
		callCount := 0

		result := Fill(length, func() int {
			callCount++
			return callCount
		})

		require.Equal(t, length, callCount)
		require.Len(t, result, length)
		require.Equal(t, length, result[length-1]) // Last element should equal the length
	})

	t.Run("creates independent slices on each call", func(t *testing.T) {
		generator := func() int { return 42 }

		slice1 := Fill(3, generator)
		slice2 := Fill(3, generator)

		// Should be equal but not the same underlying array
		require.Equal(t, slice1, slice2)
		require.NotSame(t, &slice1[0], &slice2[0], "should create independent slices")

		// Modifying one should not affect the other
		slice1[0] = 100
		require.Equal(t, 42, slice2[0], "slices should be independent")
	})

	t.Run("handles large slices efficiently", func(t *testing.T) {
		const largeSize = 100000
		result := Fill(largeSize, func() int { return 1 })

		require.Len(t, result, largeSize)
		require.Equal(t, largeSize, cap(result), "capacity should equal length for efficiency")

		// Verify all elements were set
		for i, val := range result {
			require.Equal(t, 1, val, "element at index %d should be 1", i)
		}
	})
}

// Example tests for documentation
func ExampleFill() {
	// Create 5 user IDs with generated values
	userIDs := Fill(5, func() string {
		return fmt.Sprintf("user_%d", rand.Intn(1000))
	})

	fmt.Printf("Generated %d user IDs\n", len(userIDs))
	// Output: Generated 5 user IDs
}

func ExampleFill_withStructs() {
	type User struct {
		ID   int
		Name string
	}

	counter := 0
	users := Fill(3, func() User {
		counter++
		return User{
			ID:   counter,
			Name: fmt.Sprintf("User%d", counter),
		}
	})

	fmt.Printf("Created %d users\n", len(users))
	fmt.Printf("First user: %+v\n", users[0])
	// Output:
	// Created 3 users
	// First user: {ID:1 Name:User1}
}

func ExampleFill_withRandom() {
	// Combine Fill with Random for varied test data
	outcomes := []string{"VALID", "INVALID", "EXPIRED"}

	testCases := Fill(5, func() struct {
		ID      int
		Outcome string
	} {
		return struct {
			ID      int
			Outcome string
		}{
			ID:      rand.Intn(1000),
			Outcome: Random(outcomes),
		}
	})

	fmt.Printf("Generated %d test cases\n", len(testCases))
	// Output: Generated 5 test cases
}

// Benchmark tests for Fill function
func BenchmarkFill(b *testing.B) {
	b.Run("Fill_1000_ints", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Fill(1000, func() int { return rand.Intn(1000) })
		}
	})

	b.Run("Fill_10000_strings", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Fill(10000, func() string {
				return fmt.Sprintf("item_%d", rand.Intn(10000))
			})
		}
	})

	b.Run("Fill_1000000_ints", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Fill(1000000, func() int { return i })
		}
	})
}

// Test performance comparison with manual slice creation
func BenchmarkFillVsManual(b *testing.B) {
	const size = 100000

	b.Run("Fill", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Fill(size, func() int { return rand.Intn(1000) })
		}
	})

	b.Run("Manual_make_and_loop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := make([]int, size)
			for j := 0; j < size; j++ {
				slice[j] = rand.Intn(1000)
			}
		}
	})

	b.Run("Append_based", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var slice []int
			for j := 0; j < size; j++ {
				slice = append(slice, rand.Intn(1000))
			}
		}
	})
}
