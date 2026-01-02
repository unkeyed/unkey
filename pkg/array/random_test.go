package array

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandom(t *testing.T) {
	t.Run("returns element from slice", func(t *testing.T) {
		slice := []string{"a", "b", "c", "d", "e"}
		result := Random(slice)

		require.Contains(t, slice, result)
	})

	t.Run("works with single element slice", func(t *testing.T) {
		slice := []int{42}
		result := Random(slice)

		require.Equal(t, 42, result)
	})

	t.Run("panics on empty slice", func(t *testing.T) {
		require.Panics(t, func() {
			Random([]string{})
		})
	})

	t.Run("returns different elements over multiple calls", func(t *testing.T) {
		// This test has a small chance of flaking, but with 20 different elements
		// and 100 calls, the probability of getting the same element every time
		// is astronomically small
		elements := make([]int, 20)
		for i := range elements {
			elements[i] = i
		}

		results := make(map[int]bool)
		for i := 0; i < 100; i++ {
			result := Random(elements)
			results[result] = true
		}

		// We should see at least a few different results
		require.Greater(t, len(results), 1, "should return varied results over multiple calls")
	})

	t.Run("works with different types", func(t *testing.T) {
		// Test with structs
		type Point struct{ X, Y int }
		points := []Point{{1, 2}, {3, 4}, {5, 6}}
		point := Random(points)
		require.Contains(t, points, point)

		// Test with booleans
		bools := []bool{true, false}
		result := Random(bools)
		require.Contains(t, bools, result)
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		original := []int{1, 2, 3, 4, 5}
		originalCopy := make([]int, len(original))
		copy(originalCopy, original)

		// Call Random many times
		for i := 0; i < 50; i++ {
			Random(original)
		}

		require.Equal(t, originalCopy, original, "original slice should not be modified")
	})
}

// Test the combination of Fill and Random working together
func TestFillAndRandomCombination(t *testing.T) {
	t.Run("can use Random in Fill generator", func(t *testing.T) {
		options := []string{"A", "B", "C"}

		slice := Fill(10, func() string {
			return Random(options)
		})

		// All elements should be one of the options
		for _, element := range slice {
			require.Contains(t, options, element)
		}

		// Should have some variety (this could theoretically fail but is very unlikely)
		uniqueElements := make(map[string]bool)
		for _, element := range slice {
			uniqueElements[element] = true
		}
		require.Greater(t, len(uniqueElements), 1, "should have some variety in results")
	})
}

// Example tests for documentation
func ExampleRandom() {
	outcomes := []string{"VALID", "INVALID", "EXPIRED"}

	// Select a random outcome
	outcome := Random(outcomes)

	// Verify it's one of our options
	fmt.Printf("Selected outcome is valid: %t\n",
		outcome == "VALID" || outcome == "INVALID" || outcome == "EXPIRED")
	// Output: Selected outcome is valid: true
}

// Benchmark tests for Random function
func BenchmarkRandom(b *testing.B) {
	b.Run("Random_small_slice", func(b *testing.B) {
		slice := []string{"a", "b", "c", "d", "e"}
		b.ResetTimer()

		for b.Loop() {
			Random(slice)
		}
	})

	b.Run("Random_large_slice", func(b *testing.B) {
		slice := make([]int, 1000)
		for i := range slice {
			slice[i] = i
		}
		b.ResetTimer()

		for b.Loop() {
			Random(slice)
		}
	})
}
