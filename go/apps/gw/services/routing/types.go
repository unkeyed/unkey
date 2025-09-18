package routing

import partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"

// ConfigWithWorkspace holds gateway configuration and workspace ID.
type ConfigWithWorkspace struct {
	Config      *partitionv1.GatewayConfig
	WorkspaceID string
}
