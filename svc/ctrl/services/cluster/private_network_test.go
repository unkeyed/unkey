package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type privateNetworkDatabase struct {
	db.Database
	rows   []db.ListPrivateNetworkAppsRow
	err    error
	params db.ListPrivateNetworkAppsParams
}

func (d *privateNetworkDatabase) FindCluster(context.Context, db.FindClusterParams) (db.FindClusterRow, error) {
	return db.FindClusterRow{RegionPlatform: "aws", RegionName: "eu-central-1"}, nil
}

func (d *privateNetworkDatabase) ListPrivateNetworkApps(_ context.Context, params db.ListPrivateNetworkAppsParams) ([]db.ListPrivateNetworkAppsRow, error) {
	d.params = params
	return d.rows, d.err
}

func TestPrivateNetworkSnapshotIsAuthenticatedCompleteAndPlatformScoped(t *testing.T) {
	database := &privateNetworkDatabase{rows: []db.ListPrivateNetworkAppsRow{{
		AppID: "app-a", AppSlug: "api", WorkspaceID: "workspace-a", ProjectID: "project-a",
		DeploymentID: "deployment-b", EnvironmentID: "prod-api", Port: 8080,
		CallerDeploymentID: "deployment-a", BindingID: "binding-a", BindingName: "database",
		K8sNamespace: "workspace-a",
	}}}
	clusterCache, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 1, Resource: "test_private_network_clusters", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(clusterCache.Close)
	service := &Service{db: database, bearer: "test-bearer", clusterCache: clusterCache}
	request := connect.NewRequest(&ctrlv1.GetPrivateNetworkStateRequest{
		Cluster: &ctrlv1.ClusterKey{CellId: "cell-a", Region: "eu-central-1", Platform: "aws"},
	})

	response, err := service.GetPrivateNetworkState(t.Context(), request)
	require.Nil(t, response)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	request.Header().Set("Authorization", "Bearer test-bearer")
	response, err = service.GetPrivateNetworkState(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, "aws", database.params.Platform)
	require.Equal(t, int32(privateNetworkAppsMax+1), database.params.Limit)
	require.Len(t, response.Msg.GetApps(), 1)
	require.Equal(t, "deployment-b", response.Msg.GetApps()[0].GetDeploymentId())
	require.Equal(t, "prod-api", response.Msg.GetApps()[0].GetEnvironmentId())
	require.Equal(t, "deployment-a", response.Msg.GetApps()[0].GetCallerDeploymentId())
	require.Equal(t, "binding-a", response.Msg.GetApps()[0].GetBindingId())

	database.err = errors.New("database unavailable")
	response, err = service.GetPrivateNetworkState(t.Context(), request)
	require.Nil(t, response)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))

	database.err = nil
	database.rows[0].K8sNamespace = ""
	response, err = service.GetPrivateNetworkState(t.Context(), request)
	require.Nil(t, response)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	database.rows = make([]db.ListPrivateNetworkAppsRow, privateNetworkAppsMax+1)
	response, err = service.GetPrivateNetworkState(t.Context(), request)
	require.Nil(t, response)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	database.rows = nil
	response, err = service.GetPrivateNetworkState(t.Context(), request)
	require.NoError(t, err)
	require.Empty(t, response.Msg.GetApps())
}
