package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreatePolicies authorizes creating or replacing a gateway's policy collection.
type CreatePolicies struct{}

func (CreatePolicies) ActionFor(urn.Gateway) {}
func (CreatePolicies) String() string        { return "create_policies" }

// ReadPolicy authorizes reading a gateway policy.
type ReadPolicy struct{}

func (ReadPolicy) ActionFor(urn.GatewayPolicy) {}
func (ReadPolicy) String() string              { return "read_policy" }

// UpdatePolicy authorizes updating a gateway policy.
type UpdatePolicy struct{}

func (UpdatePolicy) ActionFor(urn.GatewayPolicy) {}
func (UpdatePolicy) String() string              { return "update_policy" }
