package ratelimit

import "time"

func calculateSequence(t time.Time, duration time.Duration) int64 {
	return t.UnixMilli() / duration.Milliseconds()
}
