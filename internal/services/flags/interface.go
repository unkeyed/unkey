package flags

import "context"

// Service reports workspace feature flags. Implementations are safe for
// concurrent use.
type Service interface {
	// Values returns every defined flag of the workspace with its resolved
	// value, including flags that resolve to false. Unlike the flag methods, it
	// returns load errors instead of falling back to code defaults. The map is
	// a copy.
	Values(ctx context.Context, workspaceID string) (map[string]bool, error)
}
