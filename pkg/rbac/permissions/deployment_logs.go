package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadDeploymentLogs authorizes reading deployment logs.
type ReadDeploymentLogs struct{}

func (ReadDeploymentLogs) ActionFor(urn.DeploymentLogs) {}
func (ReadDeploymentLogs) String() string               { return "read_deployment_logs" }
