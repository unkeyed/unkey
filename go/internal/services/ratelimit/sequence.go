package ratelimit

import "time"

// calculateSequence converts a timestamp to a window sequence number for rate limiting.
// It ensures consistent window boundaries across all nodes in the cluster by aligning
// windows to fixed time boundaries based on the duration.
//
// The sequence number calculation depends on whether a createdAt timestamp is provided:
//   - If createdAt is provided: sequences are relative to that timestamp (per-identifier fairness)
//   - If createdAt is nil: sequences are relative to Unix epoch (shared boundaries)
//
// Parameters:
//   - t: The time to convert to a sequence number
//   - duration: The length of each rate limit window
//   - createdAt: Optional creation timestamp for per-identifier windows
//
// Returns:
//   - int64: A sequence number with these properties:
//   - Monotonically increasing with time
//   - Adjacent windows have consecutive numbers
//   - Same number for any time within the same window
//   - Aligned to clean time boundaries (either createdAt or epoch)
//
// Performance:
//   - O(1) time complexity
//   - No allocations
//   - Thread-safe (no state)
//
// Example Usage:
//
//	// Epoch-based (shared boundaries)
//	seq := calculateSequence(time.Now(), time.Minute, nil)
//
//	// Creation-based (per-identifier fairness)
//	createdAt := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
//	seq := calculateSequence(time.Now(), 30*24*time.Hour, &createdAt)
func calculateSequence(t time.Time, duration time.Duration, createdAt *time.Time) int64 {
	if createdAt != nil {
		// Creation-based: align windows to when the rate limit was created
		// This ensures the identifier gets the full duration
		timeSinceCreation := t.Sub(*createdAt).Milliseconds()
		return timeSinceCreation / duration.Milliseconds()
	}
	// Epoch-based: align windows to Unix epoch (traditional behavior)
	// All identifiers with same duration share window boundaries
	return t.UnixMilli() / duration.Milliseconds()
}
