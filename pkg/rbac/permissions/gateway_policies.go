package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadPolicy authorizes reading an environment's gateway policies.
//
// Valid resource: urn.GatewayPolicy.
type ReadPolicy struct{}

func (ReadPolicy) ActionFor(urn.GatewayPolicy) {}
func (ReadPolicy) String() string              { return "read_policy" }

// WritePolicy authorizes replacing an environment's entire gateway policy list
// or updating a single policy in place.
//
// Valid resource: urn.GatewayPolicy. Replacing the list uses a wildcard policy
// id because it covers policies that do not exist yet when the request is
// authorized.
type WritePolicy struct{}

func (WritePolicy) ActionFor(urn.GatewayPolicy) {}
func (WritePolicy) String() string              { return "write_policy" }
