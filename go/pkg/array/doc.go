// Package array provides generic utility functions for creating and manipulating slices with generated data.
//
// The package implements two core operations essential for test data generation and slice manipulation:
// creating slices filled with generated values, and selecting random elements from existing slices.
// Both functions leverage Go generics to provide type safety while maintaining high performance.
//
// This package was designed specifically to address common patterns in test data creation, particularly
// for scenarios requiring millions of generated data structures. Traditional approaches using append()
// or manual slice allocation and iteration result in verbose, error-prone code. The array package
// eliminates this boilerplate while providing optimal performance characteristics.
//
// The design philosophy prioritizes simplicity and performance over configurability. Rather than
// providing dozens of variants with different options, the package offers two well-designed functions
// that handle the most common use cases efficiently. This approach reduces API surface area while
// ensuring predictable behavior and performance.
//
// # Key Functions
//
// The package provides two primary functions: [Fill] for creating slices with generated data, and
// [Random] for selecting elements from existing slices. These functions work independently but are
// designed to complement each other in data generation workflows.
//
// # Usage
//
// Creating slices with generated data using [Fill]:
//
//	// Generate 1000 user IDs
//	userIDs := array.Fill(1000, func() string {
//	    return fmt.Sprintf("user_%d", rand.Intn(100000))
//	})
//
//	// Create test verification records
//	verifications := array.Fill(10000, func() Verification {
//	    return Verification{
//	        ID:        uid.New(uid.VerificationPrefix),
//	        Timestamp: time.Now().UnixMilli(),
//	        Valid:     rand.Float64() < 0.8, // 80% valid
//	    }
//	})
//
// Selecting random elements using [Random]:
//
//	outcomes := []string{"VALID", "INVALID", "EXPIRED", "RATE_LIMITED"}
//	randomOutcome := array.Random(outcomes)
//
//	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
//	selectedRegion := array.Random(regions)
//
// Combining both functions for realistic test data generation:
//
//	outcomes := []string{"VALID", "INVALID", "EXPIRED", "RATE_LIMITED"}
//	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
//
//	testData := array.Fill(1000000, func() TestCase {
//	    return TestCase{
//	        ID:      uid.New(uid.TestPrefix),
//	        Outcome: array.Random(outcomes),
//	        Region:  array.Random(regions),
//	        Latency: rand.Float64() * 500, // 0-500ms
//	    }
//	})
//
// # Performance Characteristics
//
// [Fill] operates with O(n) time complexity where n is the requested length, performing exactly
// one allocation for the slice data plus the slice header (approximately 24 bytes overhead).
// The function calls the generator function exactly n times in sequential order, making performance
// highly predictable. Memory usage is sizeof(T) * length plus slice header overhead.
//
// [Random] operates with O(1) time complexity regardless of slice size, using Go's built-in
// random number generator to select an index. No memory allocations occur during selection.
// The function is optimized for high-frequency calls in data generation scenarios.
//
// # Concurrency
//
// Both functions are safe for concurrent use when operating on different slices. [Fill] creates
// independent slices on each call, so concurrent invocations do not interfere with each other.
// The safety of [Fill] operations depends on the provided generator function - if the generator
// accesses shared state without synchronization, race conditions may occur.
//
// [Random] uses Go's global random number generator, which is protected by a mutex internally.
// Under extremely high concurrency (thousands of goroutines), this mutex may become a bottleneck.
// For such scenarios, consider using per-goroutine random number generators with [RandomWithRNG]
// if added to the package in the future.
//
// # Error Handling
//
// The package follows Go's panic-for-programming-errors convention. [Random] panics when called
// with empty slices, as this represents a programming error rather than a recoverable condition.
// This design choice eliminates the overhead of error checking in performance-critical code paths
// where empty slices should never occur.
//
// [Fill] cannot fail under normal conditions. If the generator function panics, [Fill] will
// propagate the panic. Memory exhaustion during slice allocation will result in a runtime panic,
// consistent with Go's make() function behavior.
//
// # Design Rationale
//
// The package API was designed based on analysis of common patterns in test data generation across
// large codebases. [Fill] takes a length parameter rather than a pre-allocated slice to reduce
// boilerplate and eliminate a common source of bugs (mismatched slice length and generation count).
// The single allocation approach provides optimal performance and predictable memory usage.
//
// [Random] returns elements by value rather than by reference to prevent accidental mutation of
// source slices. The uniform distribution ensures all elements have equal selection probability,
// making it suitable for unbiased test data generation.
//
// The generic design eliminates the need for type-specific variants while maintaining compile-time
// type safety. This approach provides better performance than interface{}-based alternatives and
// eliminates runtime type assertions.
package array
