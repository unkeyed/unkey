package ratelimit

import "time"

// calculateSequence converts a timestamp to a window sequence number for rate limiting.
// It ensures consistent window boundaries across all nodes in the cluster by aligning
// windows to fixed time boundaries based on the duration.
//
// The sequence number is calculated by dividing the Unix timestamp (in milliseconds)
// by the window duration (in milliseconds). This creates a monotonically increasing
// sequence where each number uniquely identifies a time window.
//
// Parameters:
//   - t: The time to convert to a sequence number
//   - duration: The length of each rate limit window
//
// Returns:
//   - int64: A sequence number with these properties:
//   - Monotonically increasing with time
//   - Adjacent windows have consecutive numbers
//   - Same number for any time within the same window
//   - Aligned to clean time boundaries (e.g., minutes)
//
// Performance:
//   - O(1) time complexity
//   - No allocations
//   - Thread-safe (no state)
//
// Example Usage:
//
//	// For a 1-minute window
//	seq := calculateSequence(time.Now(), time.Minute)
//
//	// Times within same minute get same sequence
//	t1 := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
//	t2 := time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC)
//	seq1 := calculateSequence(t1, time.Minute) // e.g., 26297340
//	seq2 := calculateSequence(t2, time.Minute) // same as seq1
//
//	// Adjacent minutes get consecutive sequences
//	t3 := time.Date(2024, 1, 1, 12, 31, 0, 0, time.UTC)
//	seq3 := calculateSequence(t3, time.Minute) // seq1 + 1
func calculateSequence(t time.Time, duration time.Duration) int64 {
	return t.UnixMilli() / duration.Milliseconds()
}
