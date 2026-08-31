package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadGatewayPolicy authorizes reading a gateway policy.
type ReadGatewayPolicy struct{}

func (ReadGatewayPolicy) ActionFor(urn.GatewayPolicy) {}
func (ReadGatewayPolicy) String() string              { return "read_gateway_policy" }

// WriteGatewayPolicy authorizes creating or updating a gateway policy.
type WriteGatewayPolicy struct{}

func (WriteGatewayPolicy) ActionFor(urn.GatewayPolicy) {}
func (WriteGatewayPolicy) String() string              { return "write_gateway_policy" }

// DeleteGatewayPolicy authorizes deleting a gateway policy.
type DeleteGatewayPolicy struct{}

func (DeleteGatewayPolicy) ActionFor(urn.GatewayPolicy) {}
func (DeleteGatewayPolicy) String() string              { return "delete_gateway_policy" }
