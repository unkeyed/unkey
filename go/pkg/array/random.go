package array

import (
	"math/rand"
	"time"
)

// Random returns a uniformly selected element from the provided slice.
//
// The function uses Go's global random number generator to select an index with uniform probability
// distribution, ensuring each element has an equal chance of selection regardless of its value,
// position, or type. The selection process does not modify the source slice in any way.
//
// Random is optimized for high-frequency selection in data generation scenarios where uniform
// distribution is required. The constant-time selection makes it suitable for use within tight
// loops or generator functions without significant performance impact.
//
// Parameters:
//   - slice: The source slice from which to select an element. Must contain at least one element.
//     The slice type can be any type T, including primitive types, structs, interfaces, or other
//     slices. The slice is not modified by the selection operation.
//
// Returns a single element from the slice, selected with uniform probability. The element is
// returned by value, creating a copy for value types and copying the reference for pointer types.
// This ensures that modifications to the returned value do not affect the original slice contents.
//
// Panics if the slice is empty (len(slice) == 0). This design prioritizes performance over error
// handling, as checking for empty slices on every call would add overhead that becomes significant
// in high-frequency usage scenarios. The panic-on-empty-slice behavior is consistent with other
// Go standard library functions that have preconditions (such as indexing operations).
//
// Behavior details:
//   - Selection uses uniform probability distribution - each element has probability 1/len(slice)
//   - Random number generation uses Go's global PRNG, automatically seeded at package initialization
//   - The source slice is never modified, even temporarily
//   - Returned values are independent copies (for value types) or independent references (for pointer types)
//   - Multiple calls with the same slice will typically return different elements
//
// Performance characteristics:
//   - Time complexity: O(1) constant time regardless of slice length
//   - Space complexity: O(1) no additional memory allocated for selection process
//   - Memory allocation: None - selection operates entirely on existing data
//   - Randomness quality: Uses Go's default PRNG with period suitable for typical use cases
//
// The constant-time performance makes Random suitable for use in performance-critical code paths,
// including tight loops generating millions of data points. The uniform distribution ensures
// unbiased selection, making it appropriate for statistical sampling and fair randomization.
//
// Concurrency considerations:
//
//	Random is safe for concurrent use from multiple goroutines, but may experience contention
//	under extreme concurrency due to Go's global random number generator mutex. For applications
//	with hundreds of goroutines making continuous Random calls, the mutex contention may become
//	a bottleneck. In such cases, consider using per-goroutine random number generators.
//
// The global random number generator is seeded automatically when the package is imported,
// using the current nanosecond timestamp. This ensures different behavior across program runs
// without requiring explicit seeding by callers.
//
// Randomness quality and suitability:
//
//	Random uses Go's default pseudorandom number generator (currently based on a linear
//	congruential generator), which provides statistical properties adequate for testing,
//	simulation, and non-cryptographic randomization. The generator is NOT suitable for
//	cryptographic applications, security-sensitive randomization, or scenarios requiring
//	cryptographically secure random numbers.
//
// Error conditions:
//
//	The only error condition is an empty slice, which results in a panic. This represents a
//	programming error that should be detected during development rather than handled at runtime.
//	Callers must ensure slices are non-empty before calling Random.
//
// Edge cases and special scenarios:
//   - Single element slice: Always returns the single element, no randomness involved
//   - Very large slices: Performance remains constant regardless of slice size
//   - Slices containing nil pointers: May return nil, which is often valid depending on use case
//   - Slices with duplicate elements: Each position has equal selection probability, duplicates may be selected multiple times
//
// Common usage patterns:
//
//	// Select random test outcome
//	outcomes := []string{"VALID", "INVALID", "EXPIRED", "RATE_LIMITED", "DISABLED"}
//	randomOutcome := array.Random(outcomes)
//
//	// Choose random server region for testing
//	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "ap-northeast-1"}
//	testRegion := array.Random(regions)
//
//	// Select random configuration for each test case
//	timeouts := []time.Duration{1*time.Second, 5*time.Second, 30*time.Second, 60*time.Second}
//	for i := 0; i < 1000; i++ {
//	    testCases[i].Timeout = array.Random(timeouts)
//	}
//
//	// Random selection in data generation pipelines
//	userTypes := []UserType{Premium, Standard, Trial, Enterprise}
//	users := array.Fill(10000, func() User {
//	    return User{
//	        ID:   uid.New(uid.UserPrefix),
//	        Type: array.Random(userTypes),
//	        CreatedAt: time.Now().Add(-time.Duration(rand.Intn(365*24)) * time.Hour),
//	    }
//	})
//
// Integration with other array functions:
//
//	Random is designed to complement [Fill] for comprehensive data generation workflows. The
//	combination allows creation of large datasets with realistic variation and distribution
//	patterns. Random can be called from within Fill generator functions without performance
//	concerns due to its constant-time operation.
//
// Anti-patterns to avoid:
//
//	// Bad: Empty slice check adds unnecessary overhead in tight loops
//	func selectSafely(slice []string) string {
//	    if len(slice) == 0 {
//	        return ""  // Better to fix the calling code
//	    }
//	    return array.Random(slice)
//	}
//
//	// Good: Ensure slices are non-empty at construction time
//	outcomes := []string{"VALID", "INVALID"} // Always has elements
//	outcome := array.Random(outcomes)        // Safe to call
//
//	// Bad: Recreating selection slice repeatedly
//	for i := 0; i < 1000000; i++ {
//	    options := []string{"A", "B", "C"}  // Wasteful allocation
//	    result[i] = array.Random(options)
//	}
//
//	// Good: Reuse selection slice
//	options := []string{"A", "B", "C"}
//	for i := 0; i < 1000000; i++ {
//	    result[i] = array.Random(options)
//	}
func Random[T any](slice []T) T {
	if len(slice) == 0 {
		panic("cannot select random element from empty slice")
	}
	return slice[rand.Intn(len(slice))]
}

func init() {
	// Seed the global random number generator with current nanosecond timestamp to ensure
	// different behavior across program runs. The nanosecond precision provides sufficient
	// entropy for typical testing and simulation use cases without requiring external
	// entropy sources or cryptographic seeding.
	rand.Seed(time.Now().UnixNano())
}
