package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

const privateNetworkPageSize = 1000

// StreamPrivateNetworkState sends the complete private network snapshot for
// the caller's platform. Every page is read in one transaction, so the pages
// form one consistent snapshot, and the snapshot is sent after the
// transaction ends so a slow reader never holds it open. Krane treats a stream
// without the final complete chunk as partial.
func (s *Service) StreamPrivateNetworkState(ctx context.Context, req *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
	result := "success"
	defer func() { metrics.PrivateNetworkSnapshotsTotal.WithLabelValues(result).Inc() }()

	if err := auth.Authenticate(req, s.bearer); err != nil {
		result = "unauthenticated"
		return err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		result = "unknown_cluster"
		if connect.CodeOf(err) == connect.CodeInternal {
			result = "database_error"
		}
		return err
	}

	readStarted := time.Now()
	connections, err := db.TxWithResult(ctx, s.db.RO(), func(txCtx context.Context, tx db.DBTX) ([]*ctrlv1.PrivateNetworkConnection, error) {
		queries := db.NewQueries(tx)
		connections, err := listPrivateNetworkConnections(txCtx, queries, cluster.RegionPlatform)
		if err != nil {
			return nil, err
		}

		replicas, err := listPrivateNetworkReplicas(txCtx, queries, cluster.RegionPlatform)
		if err != nil {
			return nil, err
		}

		return append(connections, replicas...), nil
	})
	metrics.PrivateNetworkSnapshotReadDurationSeconds.Observe(time.Since(readStarted).Seconds())
	if err != nil {
		result = "database_error"
		return connect.NewError(connect.CodeInternal, err)
	}

	for start := 0; start < len(connections); start += privateNetworkPageSize {
		chunk := &ctrlv1.PrivateNetworkStateChunk{Connections: connections[start:min(start+privateNetworkPageSize, len(connections))]}
		if err := stream.Send(chunk); err != nil {
			result = "send_error"
			return err
		}
	}

	if err := stream.Send(&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: uint64(len(connections))}); err != nil {
		result = "send_error"
		return err
	}
	return nil
}

func listPrivateNetworkConnections(ctx context.Context, queries *db.Queries, platform string) ([]*ctrlv1.PrivateNetworkConnection, error) {
	var connections []*ctrlv1.PrivateNetworkConnection
	params := db.ListPrivateNetworkConnectionsParams{AfterPk: 0, AfterCallerDeploymentID: "", Platform: platform, Limit: privateNetworkPageSize}
	for {
		rows, err := queries.ListPrivateNetworkConnections(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			connections = append(connections, &ctrlv1.PrivateNetworkConnection{
				WorkspaceId:         row.WorkspaceID,
				ProjectId:           row.ProjectID,
				TargetAppId:         row.AppID,
				TargetAppSlug:       row.AppSlug,
				K8SNamespace:        row.K8sNamespace,
				TargetDeploymentId:  row.DeploymentID,
				TargetPort:          row.Port,
				TargetEnvironmentId: row.EnvironmentID,
				CallerDeploymentId:  row.CallerDeploymentID,
				ConnectionId:        row.ConnectionID,
				ConnectionName:      row.ConnectionName,
			})
		}

		if len(rows) < privateNetworkPageSize {
			return connections, nil
		}
		last := rows[len(rows)-1]
		params.AfterPk, params.AfterCallerDeploymentID = last.Pk, last.CallerDeploymentID
	}
}

func listPrivateNetworkReplicas(ctx context.Context, queries *db.Queries, platform string) ([]*ctrlv1.PrivateNetworkConnection, error) {
	var connections []*ctrlv1.PrivateNetworkConnection
	params := db.ListPrivateNetworkReplicasParams{AfterDeploymentID: "", Platform: platform, Limit: privateNetworkPageSize}
	for {
		rows, err := queries.ListPrivateNetworkReplicas(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			if _, ok := appconnection.ReplicaHost(row.AppSlug); !ok {
				continue
			}
			connections = append(connections, &ctrlv1.PrivateNetworkConnection{
				WorkspaceId:         row.WorkspaceID,
				ProjectId:           row.ProjectID,
				TargetAppId:         row.AppID,
				TargetAppSlug:       row.AppSlug,
				K8SNamespace:        row.K8sNamespace,
				TargetDeploymentId:  row.DeploymentID,
				TargetPort:          row.Port,
				TargetEnvironmentId: row.EnvironmentID,
				CallerDeploymentId:  row.DeploymentID,
				ConnectionId:        "self-" + row.DeploymentID,
				ConnectionName:      row.AppSlug,
			})
		}

		if len(rows) < privateNetworkPageSize {
			return connections, nil
		}
		params.AfterDeploymentID = rows[len(rows)-1].DeploymentID
	}
}
