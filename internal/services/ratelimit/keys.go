package ratelimit

import "fmt"

// counterKey identifies a single sliding-window counter in memory.
// Used as a sync.Map key — struct comparison is allocation-free.
type counterKey struct {
	workspaceID string
	namespace   string
	identifier  string
	durationMs  int64
	sequence    int64
}

// redisKey returns the Redis key for this counter in the origin backend.
// Format: "{workspaceID}-{namespace}-{identifier}-{durationMs}:{sequence}".
// WorkspaceID is part of the key so two workspaces using the same namespace
// string never share counter state.
func (k counterKey) redisKey() string {
	return fmt.Sprintf("%s-%s-%s-%d:%d", k.workspaceID, k.namespace, k.identifier, k.durationMs, k.sequence)
}

// strictKey identifies the strict-enforcement deadline for a (workspace,
// namespace, identifier, duration) tuple. It is NOT scoped to a window
// sequence because a deadline set in window N can extend into window N+1.
type strictKey struct {
	workspaceID string
	namespace   string
	identifier  string
	durationMs  int64
}
