package util

import (
	"math"
	"math/rand"
	"time"
)

func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// Retry a function with exponential backoff and jitter
//
// Much smarkter people than me came up with this
// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
func Retry(fn func() error) error {
	retries := 5
	base := int64(100) // ms

	cap := int64(1000) // ms
	backoff := func(i int) time.Duration {
		temp := min(cap, base*int64(math.Pow(2, float64(i))))

		sleep := temp/2 + rand.Int63n(temp/2)
		sleep = min(cap, base+rand.Int63n(sleep*3-base))

		return time.Duration(sleep) * time.Millisecond

	}

	var err error
	for i := 0; i < retries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backoff(i))
	}
	return err

}
