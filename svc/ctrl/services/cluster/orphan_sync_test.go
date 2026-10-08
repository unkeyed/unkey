package cluster

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestFullSyncRetriesMissingDeploymentUntilRegionalConfirmation(t *testing.T) {
	f := newDeletionFixture(t, 2)
	f.createInstance(t, f.regions[0])
	require.NoError(t, f.database.DeleteDeploymentsByEnvironmentId(t.Context(), f.environmentID))
	_, handler := ctrlv1connect.NewClusterServiceHandler(f.service)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := ctrlv1connect.NewClusterServiceClient(server.Client(), server.URL)
	for attempt := 0; attempt < 2; attempt++ {
		req := connect.NewRequest(&ctrlv1.SyncDesiredStateRequest{Cluster: f.clusterKey})
		req.Header().Set("Authorization", "Bearer test-token")
		stream, err := client.SyncDesiredState(t.Context(), req)
		require.NoError(t, err)
		require.True(t, stream.Receive())
		removal := stream.Msg().GetDeployment().GetDelete()
		require.Equal(t, f.deployment.ID, removal.GetDeploymentId())
		require.True(t, removal.GetPermanent())
		require.NotEmpty(t, removal.GetK8SNamespace())
		require.False(t, stream.Receive())
		require.NoError(t, stream.Err())
		require.NoError(t, stream.Close())
	}
	require.NoError(t, db.Tx(t.Context(), f.database.RW(), func(ctx context.Context, tx db.DBTX) error {
		return confirmDeploymentRemoval(ctx, db.NewQueries(tx), f.regions[0].ID, &ctrlv1.ReportDeploymentStatusRequest_Delete{
			DeploymentId: f.deployment.ID, RemovalConfirmed: true,
		})
	}))
	for i, region := range f.regions {
		rows, err := f.database.ListOrphanedDeploymentTopologiesByRegion(t.Context(), db.ListOrphanedDeploymentTopologiesByRegionParams{
			RegionID: region.ID, Limit: 10,
		})
		require.NoError(t, err)
		require.Len(t, rows, i)
	}
	instances, err := f.database.FindInstancesByDeploymentId(t.Context(), f.deployment.ID)
	require.NoError(t, err)
	require.Empty(t, instances)
}
