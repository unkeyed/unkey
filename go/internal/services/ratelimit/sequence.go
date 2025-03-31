package ratelimit

import "time"

// calculateSequence converts a timestamp to a window sequence number.
// The sequence number uniquely identifies a time window and is calculated
// by dividing the Unix timestamp in milliseconds by the window duration.
//
// This approach ensures:
// - Windows align on clean time boundaries for all durations
// - Sequence numbers are monotonically increasing
// - Adjacent windows have consecutive sequence numbers
func calculateSequence(t time.Time, duration time.Duration) int64 {
	return t.UnixMilli() / duration.Milliseconds()
}
