package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/privatenetwork"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const privateNetworkPageSize = 1000

// StreamPrivateNetworkState sends the complete private network snapshot for
// the caller's platform. Every page is read in one transaction, so the pages
// form one consistent snapshot, and the snapshot is sent after the
// transaction ends so a slow reader never holds it open. Krane treats a stream
// without the final complete chunk as partial.
func (s *Service) StreamPrivateNetworkState(ctx context.Context, req *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		return err
	}

	apps, err := db.TxWithResult(ctx, s.db.RO(), func(txCtx context.Context, tx db.DBTX) ([]*ctrlv1.PrivateNetworkApp, error) {
		queries := db.NewQueries(tx)
		bindings, err := listPrivateNetworkBindings(txCtx, queries, cluster.RegionPlatform)
		if err != nil {
			return nil, err
		}

		replicas, err := listPrivateNetworkReplicas(txCtx, queries, cluster.RegionPlatform)
		if err != nil {
			return nil, err
		}

		return append(bindings, replicas...), nil
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for start := 0; start < len(apps); start += privateNetworkPageSize {
		chunk := &ctrlv1.PrivateNetworkStateChunk{Apps: apps[start:min(start+privateNetworkPageSize, len(apps))]}
		if err := stream.Send(chunk); err != nil {
			return err
		}
	}

	return stream.Send(&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: uint64(len(apps))})
}

func listPrivateNetworkBindings(ctx context.Context, queries *db.Queries, platform string) ([]*ctrlv1.PrivateNetworkApp, error) {
	var apps []*ctrlv1.PrivateNetworkApp
	params := db.ListPrivateNetworkBindingsParams{AfterPk: 0, AfterCallerDeploymentID: "", Platform: platform, Limit: privateNetworkPageSize}
	for {
		rows, err := queries.ListPrivateNetworkBindings(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
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

		if len(rows) < privateNetworkPageSize {
			return apps, nil
		}
		last := rows[len(rows)-1]
		params.AfterPk, params.AfterCallerDeploymentID = last.Pk, last.CallerDeploymentID
	}
}

func listPrivateNetworkReplicas(ctx context.Context, queries *db.Queries, platform string) ([]*ctrlv1.PrivateNetworkApp, error) {
	var apps []*ctrlv1.PrivateNetworkApp
	params := db.ListPrivateNetworkReplicasParams{AfterDeploymentID: "", Platform: platform, Limit: privateNetworkPageSize}
	for {
		rows, err := queries.ListPrivateNetworkReplicas(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			if _, ok := privatenetwork.ReplicaHost(row.AppSlug); !ok {
				continue
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
				CallerDeploymentId: row.DeploymentID,
				BindingId:          "self-" + row.DeploymentID,
				BindingName:        row.AppSlug,
			})
		}

		if len(rows) < privateNetworkPageSize {
			return apps, nil
		}
		params.AfterDeploymentID = rows[len(rows)-1].DeploymentID
	}
}
