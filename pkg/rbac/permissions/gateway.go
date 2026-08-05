package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadPolicy authorizes reading a gateway policy.
type ReadPolicy struct{}

func (ReadPolicy) ActionFor(urn.GatewayPolicy) {}
func (ReadPolicy) String() string              { return "read_policy" }

// UpdatePolicy authorizes updating a gateway policy.
type UpdatePolicy struct{}

func (UpdatePolicy) ActionFor(urn.GatewayPolicy) {}
func (UpdatePolicy) String() string              { return "update_policy" }

// SetPolicies authorizes replacing a gateway's complete policy collection.
type SetPolicies struct{}

func (SetPolicies) ActionFor(urn.Gateway) {}
func (SetPolicies) String() string        { return "set_policies" }
