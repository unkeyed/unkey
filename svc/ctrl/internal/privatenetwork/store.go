package privatenetwork

import (
	"context"
	"slices"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	callerChunkSize    = 250
	connectionPageSize = 1000
	transactionTimeout = 15 * time.Second
)

type caller struct {
	workspaceID string
	appID       string
	replica     *ctrlv1.PrivateNetworkConnection
	connections []*ctrlv1.PrivateNetworkConnection
}

type store interface {
	activeCallers(ctx context.Context, platform string, deploymentIDs []string) (map[string]caller, error)
	deploymentApps(ctx context.Context, deploymentIDs []string) ([]string, error)
	topology(ctx context.Context, platform string) (*ctrlv1.PrivateNetworkTopology, error)
}

// dbStore reads from the primary. CDC events come from the primary, and a
// replica read could miss the change an event reports.
type dbStore struct {
	database db.Database
}

func (s dbStore) activeCallers(ctx context.Context, platform string, deploymentIDs []string) (map[string]caller, error) {
	callers := make(map[string]caller)
	for chunk := range slices.Chunk(deploymentIDs, callerChunkSize) {
		if err := s.readCallers(ctx, platform, chunk, callers); err != nil {
			return nil, err
		}
	}
	return callers, nil
}

func (s dbStore) readCallers(ctx context.Context, platform string, deploymentIDs []string, callers map[string]caller) error {
	ctx, cancel := context.WithTimeout(ctx, transactionTimeout)
	defer cancel()
	return db.Tx(ctx, s.database.RW(), func(txCtx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		rows, err := queries.ListPrivateNetworkReplicas(txCtx, db.ListPrivateNetworkReplicasParams{DeploymentIds: deploymentIDs, Platform: platform})
		if err != nil || len(rows) == 0 {
			return err
		}
		active := make([]string, 0, len(rows))
		for _, row := range rows {
			active = append(active, row.DeploymentID)
			callers[row.DeploymentID] = caller{workspaceID: row.WorkspaceID, appID: row.AppID, replica: replicaFromRow(row), connections: nil}
		}

		params := db.ListPrivateNetworkConnectionsParams{
			CallerDeploymentIds: active, AfterCallerDeploymentID: "", AfterConnectionID: "",
			Limit: connectionPageSize, Platform: platform,
		}
		for {
			page, err := queries.ListPrivateNetworkConnections(txCtx, params)
			if err != nil {
				return err
			}
			for _, row := range page {
				entry := callers[row.CallerDeploymentID]
				entry.connections = append(entry.connections, connectionFromRow(row))
				callers[row.CallerDeploymentID] = entry
			}
			if len(page) < connectionPageSize {
				return nil
			}
			last := page[len(page)-1]
			params.AfterCallerDeploymentID, params.AfterConnectionID = last.CallerDeploymentID, last.ConnectionID
		}
	})
}

func (s dbStore) deploymentApps(ctx context.Context, deploymentIDs []string) ([]string, error) {
	var apps []string
	for chunk := range slices.Chunk(deploymentIDs, callerChunkSize) {
		rows, err := db.NewQueries(s.database.RW()).ListDeploymentAppIds(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			apps = append(apps, row.AppID)
		}
	}
	return apps, nil
}

func (s dbStore) topology(ctx context.Context, platform string) (*ctrlv1.PrivateNetworkTopology, error) {
	rows, err := db.NewQueries(s.database.RW()).ListPrivateNetworkClusters(ctx, platform)
	if err != nil {
		return nil, err
	}
	topology := &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: make([]*ctrlv1.ClusterKey, 0, len(rows))}
	for _, row := range rows {
		topology.Clusters = append(topology.Clusters, &ctrlv1.ClusterKey{
			Platform: row.Platform, Region: row.Region, CellId: row.CellID.String,
		})
	}
	return topology, nil
}

func connectionFromRow(row db.ListPrivateNetworkConnectionsRow) *ctrlv1.PrivateNetworkConnection {
	return &ctrlv1.PrivateNetworkConnection{
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
	}
}

func replicaFromRow(row db.ListPrivateNetworkReplicasRow) *ctrlv1.PrivateNetworkConnection {
	if _, ok := privatenetwork.ReplicaHost(row.AppSlug); !ok {
		return nil
	}
	return &ctrlv1.PrivateNetworkConnection{
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
	}
}
