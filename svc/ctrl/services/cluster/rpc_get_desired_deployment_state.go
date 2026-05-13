package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// GetDesiredDeploymentState returns the target state for a single deployment in the caller's
// region. This is a point query alternative to [Service.WatchDeployments] for cases where
// an agent needs to fetch state for a specific deployment rather than streaming all changes.
//
// The response contains either an ApplyDeployment (for running state) or DeleteDeployment
// (for archived or standby states) based on the deployment_topology's desired_state in the
// database. Unhandled desired states result in CodeInternal.
//
// Returns CodeUnauthenticated if bearer token is invalid, CodeInvalidArgument if
// region is missing, CodeNotFound if no deployment exists with the given ID in
// the specified region, or CodeInternal for database errors or unhandled states.
func (s *Service) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {

	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	region, err := s.resolveRegion(ctx, req.Msg.GetRegion())
	if err != nil {
		return nil, err
	}

	row, err := db.Query.FindDeploymentTopologyByDeploymentAndRegion(ctx, s.db.RO(), db.FindDeploymentTopologyByDeploymentAndRegionParams{
		DeploymentID: req.Msg.GetDeploymentId(),
		RegionID:     region.ID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	state, err := deploymentRowToState(deploymentRow{
		dt:              row.DeploymentTopology,
		d:               row.Deployment,
		k8sNamespace:    row.K8sNamespace,
		environmentSlug: row.EnvironmentSlug,
		regionName:      row.RegionName,
		gitRepo:         row.GitRepo,
	}, 0)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(state), nil
}
