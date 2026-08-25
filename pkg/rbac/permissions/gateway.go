package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadPolicy authorizes reading gateway policy resources.
//
// Valid resource: urn.GatewayPolicy.
type ReadPolicy struct{}

func (ReadPolicy) ActionFor(urn.GatewayPolicy) {}
func (ReadPolicy) String() string              { return "read_policy" }

// WritePolicy authorizes creating, replacing, updating, or deleting gateway
// policy resources.
//
// Valid resource: urn.GatewayPolicy.
type WritePolicy struct{}

func (WritePolicy) ActionFor(urn.GatewayPolicy) {}
func (WritePolicy) String() string              { return "write_policy" }
