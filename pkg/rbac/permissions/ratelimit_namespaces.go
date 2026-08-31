package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// SetOverride authorizes setting overrides in a rate limit namespace.
//
// Valid resource: urn.RatelimitNamespace.
type SetOverride struct{}

func (SetOverride) ActionFor(urn.RatelimitNamespace) {}
func (SetOverride) String() string                   { return "set_override" }
