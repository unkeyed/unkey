// Package array provides generic utility functions for slice generation and manipulation.
//
// The package implements core operations for test data generation: creating slices with
// generated values ([Fill]), selecting random elements ([Random]), transforming slices ([Map]),
// and aggregating slice data ([Reduce]).
//
// All functions use Go generics for type safety and are optimized for high-frequency use
// in test data generation scenarios.
//
// # Usage
//
// Generate slices with [Fill]:
//
//	userIDs := array.Fill(1000, func() string {
//	    return fmt.Sprintf("user_%d", rand.Intn(100000))
//	})
//
// Select random elements with [Random]:
//
//	outcomes := []string{"VALID", "INVALID", "EXPIRED"}
//	outcome := array.Random(outcomes)
//
// Transform data with [Map]:
//
//	numbers := []int{1, 2, 3}
//	strings := array.Map(numbers, strconv.Itoa)
//
// Aggregate data with [Reduce]:
//
//	sum := array.Reduce(numbers, func(acc, val int) int { return acc + val }, 0)
//
// # Performance
//
// [Fill] uses single allocation with O(n) time complexity. [Random] operates in O(1) constant time.
// [Map] and [Reduce] both use O(n) time with single allocation where applicable.
//
// # Concurrency
//
// All functions are safe for concurrent use when operating on different slices. [Random] uses
// Go's global RNG which may become a bottleneck under extreme concurrency.
package array
