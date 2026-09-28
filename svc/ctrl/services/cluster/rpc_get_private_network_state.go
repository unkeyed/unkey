package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const privateNetworkAppsMax = 10000

// GetPrivateNetworkState returns a complete platform-wide private network snapshot.
// A partial snapshot must never reach Krane: omission authorizes resource deletion.
func (s *Service) GetPrivateNetworkState(ctx context.Context, req *connect.Request[ctrlv1.GetPrivateNetworkStateRequest]) (*connect.Response[ctrlv1.GetPrivateNetworkStateResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		return nil, err
	}

	rows, err := s.db.ListPrivateNetworkApps(ctx, db.ListPrivateNetworkAppsParams{
		Platform: cluster.RegionPlatform,
		Limit:    privateNetworkAppsMax + 1,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(rows) > privateNetworkAppsMax {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("private network snapshot exceeds %d bindings", privateNetworkAppsMax))
	}

	apps := make([]*ctrlv1.PrivateNetworkApp, 0, len(rows))
	for _, row := range rows {
		if row.K8sNamespace == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("private network app %s has no Kubernetes namespace", row.AppID))
		}

		apps = append(apps, &ctrlv1.PrivateNetworkApp{
			WorkspaceId:        row.WorkspaceID,
			ProjectId:          row.ProjectID,
			AppId:              row.AppID,
			AppSlug:            row.AppSlug,
			K8SNamespace:       row.K8sNamespace,
			DeploymentId:       row.DeploymentID,
			Port:               row.Port,
			EnvironmentId:      row.EnvironmentID,
			CallerDeploymentId: row.CallerDeploymentID,
			BindingId:          row.BindingID,
			BindingName:        row.BindingName,
		})
	}

	return connect.NewResponse(&ctrlv1.GetPrivateNetworkStateResponse{Apps: apps}), nil
}
