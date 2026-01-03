package util

import (
	"fmt"
	"time"
)

// Retry retries the given function until it succeeds or all retries are exhausted
func Retry(fn func() error, attempts int, backoff func(n int) time.Duration) error {
	if attempts < 1 {
		return fmt.Errorf("attempts must be greater than 0")
	}

	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backoff(i))
	}
	return err

}
