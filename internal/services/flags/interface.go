package flags

import "context"

type Flag struct {
	Enabled     bool
	Description string
	AllowOptIn  bool
	AllowOptOut bool
}

// Service reports workspace feature flags. Implementations are safe for
// concurrent use.
type Service interface {
	// Values returns every defined flag of the workspace with its resolved
	// value and public enrollment metadata, including flags that resolve to false.
	// Unlike the flag methods, it returns load errors instead of falling back to
	// code defaults. The map is a copy.
	Values(ctx context.Context, workspaceID string) (map[string]Flag, error)

	// Invalidate evicts the workspace's flags from this service's local cache.
	// Call it only after a mutation commits.
	Invalidate(ctx context.Context, workspaceID string)

	// IsPrivateNetworkingEnabled reports whether new deployments in the
	// workspace join private networking. It is false unless the database
	// enables it.
	IsPrivateNetworkingEnabled(ctx context.Context, workspaceID string) bool

	// IsRatelimitGlobalCounterEnabled reports whether the workspace's rate limits
	// are counted through the cross-region global counter instead of the regional
	// counters. It is false unless the database enables it.
	IsRatelimitGlobalCounterEnabled(ctx context.Context, workspaceID string) bool
}
