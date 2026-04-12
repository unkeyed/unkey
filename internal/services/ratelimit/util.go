package ratelimit

import "fmt"

// counterKey identifies a single sliding window counter.
// Used as a sync.Map key — struct comparison is allocation-free.
type counterKey struct {
	name       string
	identifier string
	durationMs int64
	sequence   int64
}

// redisKey returns the Redis key for this counter.
// Format matches the legacy bucketKey format: "{name}-{identifier}-{durationMs}:{sequence}".
func redisKey(k counterKey) string {
	return fmt.Sprintf("%s-%s-%d:%d", k.name, k.identifier, k.durationMs, k.sequence)
}
