package ratelimit

import (
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
)

func newWindow(sequence int64, t time.Time, duration time.Duration) *ratelimitv1.Window {
	return &ratelimitv1.Window{
		Sequence: sequence,
		Start:    t.Truncate(duration).UnixMilli(),
		Duration: duration.Milliseconds(),
		Counter:  0,
	}
}
