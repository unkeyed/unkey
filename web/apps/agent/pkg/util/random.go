package util

import (
	"math/rand"
)

// RandomElement returns a random element from the given slice.
//
// If the slice is empty, it returns the zero value of the element type.
func RandomElement[T any](s []T) T {

	if len(s) == 0 {
		var t T
		return t
	}
	return s[rand.Intn(len(s))]
}
