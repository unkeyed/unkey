package array

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReduce(t *testing.T) {
	t.Run("sums integers", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5}
		sum := Reduce(numbers, func(acc, val int) int {
			return acc + val
		}, 0)

		require.Equal(t, 15, sum)
	})

	t.Run("finds maximum value", func(t *testing.T) {
		values := []float64{3.14, 2.71, 1.41, 4.67, 2.23}
		maxVal := Reduce(values, func(acc, val float64) float64 {
			if val > acc {
				return val
			}
			return acc
		}, math.Inf(-1))

		require.Equal(t, 4.67, maxVal)
	})

	t.Run("finds minimum value", func(t *testing.T) {
		values := []int{5, 3, 8, 1, 9, 2, 7}
		minVal := Reduce(values, func(acc, val int) int {
			if val < acc {
				return val
			}
			return acc
		}, math.MaxInt)

		require.Equal(t, 1, minVal)
	})

	t.Run("concatenates strings", func(t *testing.T) {
		words := []string{"hello", "world", "from", "go"}
		sentence := Reduce(words, func(acc, word string) string {
			if acc == "" {
				return word
			}
			return acc + " " + word
		}, "")

		require.Equal(t, "hello world from go", sentence)
	})

	t.Run("builds frequency map", func(t *testing.T) {
		words := []string{"hello", "world", "hello", "go", "world", "hello"}
		freq := Reduce(words, func(acc map[string]int, word string) map[string]int {
			acc[word]++
			return acc
		}, make(map[string]int))

		expected := map[string]int{"hello": 3, "world": 2, "go": 1}
		require.Equal(t, expected, freq)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		empty := []int{}
		callCount := 0
		result := Reduce(empty, func(acc, val int) int {
			callCount++
			return acc + val
		}, 42)

		require.Equal(t, 42, result, "should return initial value for empty slice")
		require.Equal(t, 0, callCount, "reduction function should not be called for empty slice")
	})

	t.Run("handles single element slice", func(t *testing.T) {
		single := []int{100}
		result := Reduce(single, func(acc, val int) int {
			return acc + val
		}, 10)

		require.Equal(t, 110, result) // 10 + 100
	})

	t.Run("transforms slice to different type", func(t *testing.T) {
		type User struct {
			ID   int
			Name string
		}

		users := []User{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
			{ID: 3, Name: "Charlie"},
		}

		nameMap := Reduce(users, func(acc map[int]string, user User) map[int]string {
			acc[user.ID] = user.Name
			return acc
		}, make(map[int]string))

		expected := map[int]string{1: "Alice", 2: "Bob", 3: "Charlie"}
		require.Equal(t, expected, nameMap)
	})

	t.Run("counts elements with condition", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		evenCount := Reduce(numbers, func(acc, val int) int {
			if val%2 == 0 {
				return acc + 1
			}
			return acc
		}, 0)

		require.Equal(t, 5, evenCount) // 2, 4, 6, 8, 10
	})

	t.Run("calculates product", func(t *testing.T) {
		numbers := []int{2, 3, 4, 5}
		product := Reduce(numbers, func(acc, val int) int {
			return acc * val
		}, 1)

		require.Equal(t, 120, product) // 2 * 3 * 4 * 5 = 120
	})

	t.Run("builds slice in reverse order", func(t *testing.T) {
		input := []string{"a", "b", "c", "d"}
		reversed := Reduce(input, func(acc []string, val string) []string {
			return append([]string{val}, acc...)
		}, []string{})

		expected := []string{"d", "c", "b", "a"}
		require.Equal(t, expected, reversed)
	})

	t.Run("groups elements by category", func(t *testing.T) {
		type Item struct {
			Category string
			Value    int
		}

		items := []Item{
			{Category: "A", Value: 10},
			{Category: "B", Value: 20},
			{Category: "A", Value: 30},
			{Category: "C", Value: 40},
			{Category: "B", Value: 50},
		}

		groups := Reduce(items, func(acc map[string][]int, item Item) map[string][]int {
			acc[item.Category] = append(acc[item.Category], item.Value)
			return acc
		}, make(map[string][]int))

		expected := map[string][]int{
			"A": {10, 30},
			"B": {20, 50},
			"C": {40},
		}
		require.Equal(t, expected, groups)
	})

	t.Run("reduction function is called exactly once per element", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		callCount := 0
		processedValues := make([]int, 0)
		accumulatorValues := make([]int, 0)

		result := Reduce(input, func(acc, val int) int {
			callCount++
			processedValues = append(processedValues, val)
			accumulatorValues = append(accumulatorValues, acc)
			return acc + val
		}, 10)

		require.Equal(t, len(input), callCount, "function should be called exactly once per element")
		require.Equal(t, input, processedValues, "function should be called with each element exactly once")
		require.Equal(t, []int{10, 11, 13, 16, 20}, accumulatorValues, "accumulator should pass through correctly")
		require.Equal(t, 25, result) // 10 + 1 + 2 + 3 + 4 + 5
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		original := []int{1, 2, 3, 4, 5}
		originalCopy := make([]int, len(original))
		copy(originalCopy, original)

		Reduce(original, func(acc, val int) int { return acc + val }, 0)

		require.Equal(t, originalCopy, original, "original slice should not be modified")
	})

	t.Run("works with different accumulator and element types", func(t *testing.T) {
		// Convert strings to total character count
		words := []string{"hello", "world", "go"}
		totalChars := Reduce(words, func(acc int, word string) int {
			return acc + len(word)
		}, 0)

		// "hello"=5, "world"=5, "go"=2, so 5+5+2=12
		require.Equal(t, 12, totalChars) // 5 + 5 + 2 = 12
	})

	t.Run("maintains order dependency", func(t *testing.T) {
		// Subtraction is order-dependent
		numbers := []int{10, 3, 2}
		leftToRight := Reduce(numbers, func(acc, val int) int {
			return acc - val
		}, 20)

		require.Equal(t, 5, leftToRight) // 20 - 10 - 3 - 2 = 5

		// String concatenation is also order-dependent
		chars := []string{"a", "b", "c"}
		concatenated := Reduce(chars, func(acc, char string) string {
			return acc + char
		}, "start")

		require.Equal(t, "startabc", concatenated)
	})

	t.Run("handles complex accumulator operations", func(t *testing.T) {
		type Stats struct {
			Count int
			Sum   int
			Max   int
			Min   int
		}

		numbers := []int{5, 2, 8, 1, 9, 3}
		stats := Reduce(numbers, func(acc Stats, val int) Stats {
			newStats := Stats{
				Count: acc.Count + 1,
				Sum:   acc.Sum + val,
				Max:   acc.Max,
				Min:   acc.Min,
			}

			if val > newStats.Max {
				newStats.Max = val
			}
			if val < newStats.Min {
				newStats.Min = val
			}

			return newStats
		}, Stats{Count: 0, Sum: 0, Max: math.MinInt, Min: math.MaxInt})

		expected := Stats{Count: 6, Sum: 28, Max: 9, Min: 1}
		require.Equal(t, expected, stats)
	})

	t.Run("works with large slices efficiently", func(t *testing.T) {
		const largeSize = 100000
		input := make([]int, largeSize)
		for i := range input {
			input[i] = 1
		}

		sum := Reduce(input, func(acc, val int) int {
			return acc + val
		}, 0)

		require.Equal(t, largeSize, sum)
	})

	t.Run("handles pointer types safely", func(t *testing.T) {
		values := []*int{intPtr(10), nil, intPtr(20), intPtr(30)}
		sum := Reduce(values, func(acc int, ptr *int) int {
			if ptr == nil {
				return acc
			}
			return acc + *ptr
		}, 0)

		require.Equal(t, 60, sum) // 10 + 20 + 30 = 60
	})
}

// Test combination of Map and Reduce
func TestMapReduceCombination(t *testing.T) {
	t.Run("map then reduce pipeline", func(t *testing.T) {
		scores := []int{85, 92, 78, 96, 88}

		// Map scores to letter grades
		letterGrades := Map(scores, func(score int) string {
			if score >= 90 {
				return "A"
			}
			if score >= 80 {
				return "B"
			}
			return "C"
		})

		// Reduce to count each grade
		gradeCount := Reduce(letterGrades, func(acc map[string]int, grade string) map[string]int {
			acc[grade]++
			return acc
		}, make(map[string]int))

		expected := map[string]int{"A": 2, "B": 2, "C": 1}
		require.Equal(t, expected, gradeCount)
	})

	t.Run("reduce then map pipeline", func(t *testing.T) {
		// Group numbers by even/odd using reduce
		numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		grouped := Reduce(numbers, func(acc map[string][]int, val int) map[string][]int {
			if val%2 == 0 {
				acc["even"] = append(acc["even"], val)
			} else {
				acc["odd"] = append(acc["odd"], val)
			}
			return acc
		}, make(map[string][]int))

		// Map the group names to counts
		counts := map[string]int{}
		for group, values := range grouped {
			counts[group] = len(values)
		}

		expected := map[string]int{"even": 5, "odd": 5}
		require.Equal(t, expected, counts)
	})
}

// Example tests for documentation
func ExampleReduce() {
	numbers := []int{1, 2, 3, 4, 5}
	sum := Reduce(numbers, func(acc, val int) int {
		return acc + val
	}, 0)

	fmt.Printf("Sum: %d\n", sum)
	// Output: Sum: 15
}

func ExampleReduce_findMax() {
	values := []float64{3.14, 2.71, 4.67, 1.41}
	maxVal := Reduce(values, func(acc, val float64) float64 {
		if val > acc {
			return val
		}
		return acc
	}, math.Inf(-1))

	fmt.Printf("Maximum: %.2f\n", maxVal)
	// Output: Maximum: 4.67
}

func ExampleReduce_buildMap() {
	words := []string{"hello", "world", "hello", "go"}
	freq := Reduce(words, func(acc map[string]int, word string) map[string]int {
		acc[word]++
		return acc
	}, make(map[string]int))

	fmt.Printf("Frequency: %v\n", freq)
	// Output: Frequency: map[go:1 hello:2 world:1]
}

func ExampleReduce_joinStrings() {
	items := []string{"apple", "banana", "cherry"}
	joined := Reduce(items, func(acc, item string) string {
		if acc == "" {
			return item
		}
		return acc + ", " + item
	}, "")

	fmt.Printf("Joined: %s\n", joined)
	// Output: Joined: apple, banana, cherry
}

func ExampleReduce_typeTransformation() {
	type User struct {
		ID   int
		Name string
	}

	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	nameMap := Reduce(users, func(acc map[int]string, user User) map[int]string {
		acc[user.ID] = user.Name
		return acc
	}, make(map[int]string))

	fmt.Printf("Name map: %v\n", nameMap)
	// Output: Name map: map[1:Alice 2:Bob]
}

// Benchmark tests for Reduce function
func BenchmarkReduce(b *testing.B) {
	b.Run("Reduce_1000_sum_ints", func(b *testing.B) {
		input := make([]int, 1000)
		for i := range input {
			input[i] = i
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			Reduce(input, func(acc, val int) int { return acc + val }, 0)
		}
	})

	b.Run("Reduce_10000_build_map", func(b *testing.B) {
		input := make([]string, 10000)
		for i := range input {
			input[i] = strconv.Itoa(i % 100) // Create some duplicates
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			Reduce(input, func(acc map[string]int, val string) map[string]int {
				acc[val]++
				return acc
			}, make(map[string]int))
		}
	})

	b.Run("Reduce_100000_find_max", func(b *testing.B) {
		input := make([]int, 100000)
		for i := range input {
			input[i] = i
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			Reduce(input, func(acc, val int) int {
				if val > acc {
					return val
				}
				return acc
			}, math.MinInt)
		}
	})
}

// Test performance comparison with manual approaches
func BenchmarkReduceVsManual(b *testing.B) {
	const size = 10000
	input := make([]int, size)
	for i := range input {
		input[i] = i
	}

	b.Run("Reduce_sum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Reduce(input, func(acc, val int) int { return acc + val }, 0)
		}
	})

	b.Run("Manual_loop_sum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sum := 0
			for _, val := range input {
				sum += val
			}
		}
	})

	b.Run("Reduce_build_map", func(b *testing.B) {
		stringInput := make([]string, size)
		for i := range stringInput {
			stringInput[i] = strconv.Itoa(i % 100)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			Reduce(stringInput, func(acc map[string]int, val string) map[string]int {
				acc[val]++
				return acc
			}, make(map[string]int))
		}
	})

	b.Run("Manual_build_map", func(b *testing.B) {
		stringInput := make([]string, size)
		for i := range stringInput {
			stringInput[i] = strconv.Itoa(i % 100)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			freq := make(map[string]int)
			for _, val := range stringInput {
				freq[val]++
			}
		}
	})
}

// Test complex reduce operations
func BenchmarkReduceComplexOperations(b *testing.B) {
	type Record struct {
		ID       int
		Category string
		Value    float64
	}

	// Create test data
	records := make([]Record, 10000)
	categories := []string{"A", "B", "C", "D", "E"}
	for i := range records {
		records[i] = Record{
			ID:       i,
			Category: categories[i%len(categories)],
			Value:    float64(i % 1000),
		}
	}

	b.Run("Group_by_category", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Reduce(records, func(acc map[string][]Record, record Record) map[string][]Record {
				acc[record.Category] = append(acc[record.Category], record)
				return acc
			}, make(map[string][]Record))
		}
	})

	b.Run("Calculate_category_stats", func(b *testing.B) {
		type CategoryStats struct {
			Count int
			Sum   float64
			Avg   float64
		}

		for i := 0; i < b.N; i++ {
			stats := Reduce(records, func(acc map[string]CategoryStats, record Record) map[string]CategoryStats {
				current := acc[record.Category]
				current.Count++
				current.Sum += record.Value
				current.Avg = current.Sum / float64(current.Count)
				acc[record.Category] = current
				return acc
			}, make(map[string]CategoryStats))
			_ = stats
		}
	})
}
