package array

import (
	"math/rand"
)

// Random returns a uniformly selected element from the slice.
//
// Uses Go's global random number generator to select with uniform probability distribution.
// Each element has equal chance of selection regardless of position or value.
//
// Panics if the slice is empty.
//
//	// Select random test outcomes
//	outcomes := []string{"VALID", "INVALID", "EXPIRED"}
//	outcome := array.Random(outcomes)
//
//	// Use in data generation
//	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
//	testData := array.Fill(1000, func() TestCase {
//	    return TestCase{Region: array.Random(regions)}
//	})
func Random[T any](slice []T) T {
	if len(slice) == 0 {
		panic("cannot select random element from empty slice")
	}
	//nolint:gosec // G404: Non-cryptographic random selection is appropriate here
	return slice[rand.Intn(len(slice))]
}
