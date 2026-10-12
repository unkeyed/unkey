package flags

import "context"

type Flag struct {
	Value        bool
	Description  string
	AllowEnable  bool
	AllowDisable bool
}

// Service reports workspace feature flags. Implementations are safe for
// concurrent use.
type Service interface {
	// Values returns every defined flag of the workspace with its resolved
	// value and public enrollment metadata, including flags that resolve to false.
	// Unlike the flag methods, it returns load errors instead of falling back to
	// code defaults. The map is a copy.
	Values(ctx context.Context, workspaceID string) (map[string]Flag, error)
}
