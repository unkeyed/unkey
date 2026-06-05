// Package ops hosts internal operator RPCs that aren't part of the normal
// customer-facing deploy flow. These exist for manual recovery scenarios
// (e.g. rebuilding a deployment whose image was accidentally deleted from
// the registry). All RPCs are bearer-authenticated.
package ops

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/svc/ctrl/services/deployment"
)

// Service implements [ctrlv1connect.OpsServiceHandler].
type Service struct {
	ctrlv1connect.UnimplementedOpsServiceHandler
	deployment *deployment.Service
	bearer     string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// DeploymentService provides the shared create/deploy machinery used by
	// recovery RPCs. RebuildDeployment delegates to it rather than
	// re-implementing the lookup + insert + workflow kickoff chain.
	DeploymentService *deployment.Service

	// Bearer is the preshared token that callers must provide in the
	// Authorization header. Typically the same bearer as DeployService.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedOpsServiceHandler: ctrlv1connect.UnimplementedOpsServiceHandler{},
		deployment:                     cfg.DeploymentService,
		bearer:                         cfg.Bearer,
	}
}

var _ ctrlv1connect.OpsServiceHandler = (*Service)(nil)
