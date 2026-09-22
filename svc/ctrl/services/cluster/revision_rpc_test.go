package cluster

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestGetDesiredDeploymentState_ReturnsRevision(t *testing.T) {
	client := newRevisionRPCClient(t)
	req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{
		Cluster:      revisionTestClusterKey(),
		DeploymentId: "deploy_test",
	})
	req.Header().Set("Authorization", "Bearer test-token")

	res, err := client.GetDesiredDeploymentState(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, uint64(41), res.Msg.GetApply().GetRevision())
}

func TestSyncDesiredState_ReturnsRevision(t *testing.T) {
	client := newRevisionRPCClient(t)
	req := connect.NewRequest(&ctrlv1.SyncDesiredStateRequest{Cluster: revisionTestClusterKey()})
	req.Header().Set("Authorization", "Bearer test-token")

	stream, err := client.SyncDesiredState(t.Context(), req)
	require.NoError(t, err)
	require.True(t, stream.Receive())
	require.Equal(t, uint64(42), stream.Msg().GetDeployment().GetApply().GetRevision())
	require.False(t, stream.Receive())
	require.NoError(t, stream.Err())
	require.NoError(t, stream.Close())
}

func newRevisionRPCClient(t *testing.T) ctrlv1connect.ClusterServiceClient {
	t.Helper()

	clusterCache, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 1, Resource: "test_cluster_revision", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(clusterCache.Close)
	service := &Service{db: &revisionDatabase{}, bearer: "test-token", clusterCache: clusterCache}
	_, handler := ctrlv1connect.NewClusterServiceHandler(service)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return ctrlv1connect.NewClusterServiceClient(server.Client(), server.URL)
}

func revisionTestClusterKey() *ctrlv1.ClusterKey {
	return &ctrlv1.ClusterKey{CellId: "cell", Platform: "local", Region: "local"}
}

type revisionDatabase struct {
	db.Database
}

func (d *revisionDatabase) FindCluster(context.Context, db.FindClusterParams) (db.FindClusterRow, error) {
	return db.FindClusterRow{RegionID: "region_test"}, nil
}

func (d *revisionDatabase) FindDeploymentTopologyByDeploymentAndRegion(context.Context, db.FindDeploymentTopologyByDeploymentAndRegionParams) (db.FindDeploymentTopologyByDeploymentAndRegionRow, error) {
	return db.FindDeploymentTopologyByDeploymentAndRegionRow{
		ID: "deploy_test", Revision: 41, DesiredStatus: db.DeploymentTopologyDesiredStatusRunning,
	}, nil
}

func (d *revisionDatabase) ListAllDeploymentTopologiesByRegion(context.Context, db.ListAllDeploymentTopologiesByRegionParams) ([]db.ListAllDeploymentTopologiesByRegionRow, error) {
	return []db.ListAllDeploymentTopologiesByRegionRow{{
		TopologyPk: 1, TopologyRevision: 42, TopologyDesiredStatus: db.DeploymentTopologyDesiredStatusRunning,
		DeploymentID: "deploy_test",
	}}, nil
}
