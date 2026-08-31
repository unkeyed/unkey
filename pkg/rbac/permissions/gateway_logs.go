package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadGatewayLogs authorizes reading gateway logs.
type ReadGatewayLogs struct{}

func (ReadGatewayLogs) ActionFor(urn.GatewayLogs) {}
func (ReadGatewayLogs) String() string            { return "read_gateway_logs" }
