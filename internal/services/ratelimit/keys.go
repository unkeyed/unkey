package ratelimit

import "fmt"

// counterKey identifies a single sliding window counter in memory.
// Used as a sync.Map key — struct comparison is allocation-free.
type counterKey struct {
	name       string
	identifier string
	durationMs int64
	sequence   int64
}

// strictKey identifies the strict-enforcement deadline for a (name, identifier,
// duration) tuple. It is NOT scoped to a window sequence because a deadline
// set in window N can extend into window N+1.
type strictKey struct {
	name       string
	identifier string
	durationMs int64
}

// redisKey returns the Redis key for a counter in the origin backend.
// Format: "{name}-{identifier}-{durationMs}:{sequence}".
func redisKey(k counterKey) string {
	return fmt.Sprintf("%s-%s-%d:%d", k.name, k.identifier, k.durationMs, k.sequence)
}
