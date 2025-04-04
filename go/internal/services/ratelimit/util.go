package ratelimit

import "fmt"

func counterKey(b bucketKey, seq int64) string {
	return fmt.Sprintf("%s:%d", b.toString(), seq)
}
