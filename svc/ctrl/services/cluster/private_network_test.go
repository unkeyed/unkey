package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/privatenetwork"
)

type unavailableCatalog struct{}

func (unavailableCatalog) Snapshot(context.Context, string) (privatenetwork.Snapshot, error) {
	return privatenetwork.Snapshot{Connections: nil, Topology: nil, Version: "", ID: "", Certified: false}, privatenetwork.ErrUnavailable
}

// TestPrivateNetworkStateReportsUnavailableCatalog guarantees that Krane gets
// a retryable Unavailable instead of an empty or partial snapshot while the
// catalog is not built.
func TestPrivateNetworkStateReportsUnavailableCatalog(t *testing.T) {
	server := containers.MySQL(t)
	database, err := db.New(server.DSN, sqlcomment.Static{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	platform := strings.ToLower(uid.New("pf"))
	cell := uid.New("cell")
	seedPrivateNetworkCluster(t, seed.New(t, database, nil), "region-"+platform, platform, sql.NullString{String: cell, Valid: true})

	client := privateNetworkClient(t, &Service{db: database, bearer: "test-bearer", clusterCache: newClusterTestCache(t), privateNetwork: unavailableCatalog{}})
	_, err = receivePrivateNetworkState(t, client, cell, platform, "test-bearer", "")
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// TestPrivateNetworkStateStreamsCatalog guarantees that Krane receives the
// complete CDC-maintained catalog in bounded chunks and in a deterministic
// order, that polling an unchanged catalog keeps its version, and that a
// stopped caller disappears.
func TestPrivateNetworkStateStreamsCatalog(t *testing.T) {
	vitess := containers.Vitess(t)
	database, err := db.New(vitess.DSN, sqlcomment.Static{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	platform := strings.ToLower(uid.New("pf"))
	cell := uid.New("cell")
	seeder := seed.New(t, database, nil)
	fixture := seedPrivateNetwork(t, seeder, platform, cell)
	target := fixture.target
	secondCell, secondRegion := seedPrivateNetworkTopology(t, seeder, platform)

	catalogs, err := privatenetwork.New(privatenetwork.Config{
		Database: database, VStream: cdc.Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true}, Clock: clock.New(),
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- catalogs.Run(ctx) }()
	t.Cleanup(func() { cancel(); require.NoError(t, <-stopped) })

	client := privateNetworkClient(t, &Service{db: database, bearer: "test-bearer", clusterCache: newClusterTestCache(t), privateNetwork: catalogs})
	receive := func(knownVersion string) ([]*ctrlv1.PrivateNetworkStateChunk, error) {
		return receivePrivateNetworkState(t, client, cell, platform, "test-bearer", knownVersion)
	}

	_, err = receivePrivateNetworkState(t, client, cell, platform, "wrong-bearer", "")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = receivePrivateNetworkState(t, client, uid.New("cell"), platform, "test-bearer", "")
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	chunks, err := receive("")
	require.NoError(t, err)
	require.Len(t, chunks, 8, "connections and replicas stream as seven full-or-partial chunks and one complete chunk")
	last := chunks[len(chunks)-1]
	require.True(t, last.GetComplete())
	require.NotEmpty(t, last.GetVersion())
	require.NotEmpty(t, last.GetSnapshotId())
	require.Empty(t, last.GetConnections())
	require.Equal(t, uint32(1), last.GetTopology().GetVersion())
	require.ElementsMatch(t, []*ctrlv1.ClusterKey{
		{CellId: cell, Region: "region-" + platform, Platform: platform},
		{CellId: secondCell, Region: secondRegion, Platform: platform},
	}, last.GetTopology().GetClusters())

	var streamed []*ctrlv1.PrivateNetworkConnection
	for _, chunk := range chunks[:len(chunks)-1] {
		require.False(t, chunk.GetComplete())
		require.Nil(t, chunk.GetTopology())
		require.False(t, chunk.GetCertified(), "certification is final metadata")
		require.LessOrEqual(t, len(chunk.GetConnections()), privateNetworkChunkSize)
		streamed = append(streamed, chunk.GetConnections()...)
	}
	var want []string
	for _, caller := range slices.Sorted(slices.Values(fixture.callers)) {
		for _, connectionID := range slices.Sorted(slices.Values(fixture.connectionIDs)) {
			want = append(want, caller+"/"+connectionID)
		}
	}
	for _, deployment := range slices.Sorted(slices.Values(fixture.replicas)) {
		want = append(want, deployment+"/self-"+deployment)
	}
	got := make([]string, 0, len(streamed))
	for _, connection := range streamed {
		got = append(got, connection.GetCallerDeploymentId()+"/"+connection.GetConnectionId())
	}
	require.Equal(t, want, got, "every enrolled caller's connections, then every replica, each ordered by caller and connection so the version is deterministic")
	require.Equal(t, uint64(len(want)), last.GetTotal())

	connections, replicas := streamed[:len(streamed)-len(fixture.replicas)], streamed[len(streamed)-len(fixture.replicas):]
	for _, connection := range connections {
		require.True(t, strings.HasPrefix(connection.GetConnectionName(), "database"), "connection %s keeps its saved name", connection.GetConnectionId())
		require.Equal(t, target, connection.GetTargetDeploymentId(), "connection %s", connection.GetConnectionId())
		require.Equal(t, int32(5432), connection.GetTargetPort(), "connection %s", connection.GetConnectionId())
	}
	for _, replica := range replicas {
		require.Equal(t, replica.GetCallerDeploymentId(), replica.GetTargetDeploymentId())
		require.Equal(t, "self", replica.GetConnectionName(), "replicas resolve under the self name whatever their app slug")
	}

	unchanged, err := receive(last.GetVersion())
	require.NoError(t, err)
	require.Len(t, unchanged, 1)
	require.True(t, unchanged[0].GetComplete())
	require.True(t, unchanged[0].GetUnchanged())
	require.Equal(t, last.GetVersion(), unchanged[0].GetVersion())
	require.Equal(t, last.GetTotal(), unchanged[0].GetTotal())
	var wait sync.WaitGroup
	for range 8 {
		wait.Go(func() {
			polled, receiveErr := receive("")
			if assert.NoError(t, receiveErr) {
				assert.Equal(t, last.GetVersion(), polled[len(polled)-1].GetVersion())
			}
		})
	}
	wait.Wait()

	callerDeploymentID := fixture.callers[0]
	stoppedAt := sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}
	require.NoError(t, database.UpdateDeploymentDesiredState(t.Context(), db.UpdateDeploymentDesiredStateParams{
		DesiredState: dbtype.DeploymentsDesiredStateStopped, UpdatedAt: stoppedAt, ID: callerDeploymentID,
	}))
	require.NoError(t, database.UpdateDeploymentTopologyDesiredStatus(t.Context(), db.UpdateDeploymentTopologyDesiredStatusParams{
		DesiredStatus: db.DeploymentTopologyDesiredStatusStopped, UpdatedAt: stoppedAt,
		DeploymentID: callerDeploymentID, RegionID: fixture.regionID,
	}))
	stoppedChunks := receiveChanged(t, receive, last.GetVersion())
	stoppedLast := stoppedChunks[len(stoppedChunks)-1]
	require.True(t, stoppedLast.GetCertified(), "a snapshot right after a committed change is certified")
	remainingCallers := map[string]struct{}{}
	for _, chunk := range stoppedChunks {
		for _, candidate := range chunk.GetConnections() {
			require.NotEqual(t, callerDeploymentID, candidate.GetCallerDeploymentId())
			remainingCallers[candidate.GetCallerDeploymentId()] = struct{}{}
		}
	}
	for _, otherCaller := range fixture.callers[1:] {
		require.Contains(t, remainingCallers, otherCaller)
	}

	setTargetStatus := func(version string, status dbtype.DeploymentsStatus) (string, []string) {
		t.Helper()
		require.NoError(t, database.UpdateDeploymentStatus(t.Context(), db.UpdateDeploymentStatusParams{
			Status: status, UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}, ID: target,
		}))
		changed := receiveChanged(t, receive, version)
		selected := map[string]struct{}{}
		for _, chunk := range changed {
			for _, connection := range chunk.GetConnections() {
				if !strings.HasPrefix(connection.GetConnectionId(), "self-") {
					selected[connection.GetTargetDeploymentId()] = struct{}{}
				}
			}
		}
		return changed[len(changed)-1].GetVersion(), slices.Sorted(maps.Keys(selected))
	}
	failedVersion, failedTargets := setTargetStatus(stoppedLast.GetVersion(), dbtype.DeploymentsStatusFailed)
	require.Equal(t, []string{""}, failedTargets, "a failed target is not selected")
	_, readyTargets := setTargetStatus(failedVersion, dbtype.DeploymentsStatusReady)
	require.Equal(t, []string{target}, readyTargets, "a ready target is selected again")
}

func receiveChanged(t *testing.T, receive func(string) ([]*ctrlv1.PrivateNetworkStateChunk, error), version string) []*ctrlv1.PrivateNetworkStateChunk {
	t.Helper()
	var changed []*ctrlv1.PrivateNetworkStateChunk
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		chunks, err := receive(version)
		require.NoError(c, err)
		require.False(c, chunks[len(chunks)-1].GetUnchanged())
		changed = chunks
	}, 30*time.Second, 50*time.Millisecond)
	return changed
}

func privateNetworkClient(t *testing.T, service *Service) ctrlv1connect.ClusterServiceClient {
	t.Helper()
	_, handler := ctrlv1connect.NewClusterServiceHandler(service)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return ctrlv1connect.NewClusterServiceClient(server.Client(), server.URL)
}

func receivePrivateNetworkState(t *testing.T, client ctrlv1connect.ClusterServiceClient, cell, platform, bearer, knownVersion string) ([]*ctrlv1.PrivateNetworkStateChunk, error) {
	t.Helper()
	request := connect.NewRequest(&ctrlv1.StreamPrivateNetworkStateRequest{
		Cluster:      &ctrlv1.ClusterKey{CellId: cell, Region: "region-" + platform, Platform: platform},
		KnownVersion: knownVersion,
	})
	request.Header().Set("Authorization", "Bearer "+bearer)
	stream, err := client.StreamPrivateNetworkState(t.Context(), request)
	if err != nil {
		return nil, err
	}
	var chunks []*ctrlv1.PrivateNetworkStateChunk
	for stream.Receive() {
		chunks = append(chunks, stream.Msg())
	}
	streamErr := stream.Err()
	if closeErr := stream.Close(); streamErr == nil {
		streamErr = closeErr
	}
	return chunks, streamErr
}

func newClusterTestCache(t *testing.T) cache.Cache[clusterCacheKey, db.FindClusterRow] {
	t.Helper()
	c, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 4, Resource: "test_private_network_clusters", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c
}

type privateNetworkFixture struct {
	callers       []string
	connectionIDs []string
	replicas      []string
	target        string
	regionID      string
}

func seedPrivateNetwork(t *testing.T, seeder *seed.Seeder, platform, cell string) privateNetworkFixture {
	t.Helper()
	const callers = privateNetworkChunkSize + 1
	ctx := t.Context()
	database := seeder.DB
	region := seedPrivateNetworkCluster(t, seeder, "region-"+platform, platform, sql.NullString{String: cell, Valid: true})

	workspace := seeder.CreateWorkspace(ctx)
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspace.ID, Name: "Project", Slug: "project",
	})
	apps, environments := map[string]string{}, map[string]string{}
	for _, slug := range []string{"api", "db", "Bad_Slug"} {
		apps[slug] = seeder.CreateApp(ctx, seed.CreateAppRequest{
			ID: uid.New(uid.AppPrefix), WorkspaceID: workspace.ID, ProjectID: project.ID, Name: slug, Slug: slug,
		}).ID
		environments[slug] = seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspace.ID, ProjectID: project.ID, AppID: apps[slug], Slug: "preview",
		}).ID
	}
	// The seeder's workspace cleanup does not remove topology, so this runs first.
	t.Cleanup(func() {
		for slug, environment := range environments {
			require.NoError(t, database.DeleteDeploymentTopologiesByEnvironmentId(context.Background(), environment))
			require.NoError(t, database.DeleteDeploymentConnectionsByEnvironmentId(context.Background(), db.DeleteDeploymentConnectionsByEnvironmentIdParams{
				AppID: apps[slug], EnvironmentID: environment,
			}))
			require.NoError(t, database.DeleteEnvironmentById(context.Background(), environment))
			require.NoError(t, database.DeleteDeploymentsByEnvironmentId(context.Background(), environment))
		}
	})

	deploy := func(app string, port int32, enabled bool) string {
		t.Helper()
		deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			WorkspaceID: workspace.ID, ProjectID: project.ID, AppID: apps[app], EnvironmentID: environments[app],
			Status: dbtype.DeploymentsStatusReady, Port: port,
			Capabilities: dbtype.DeploymentCapabilities{PrivateNetworking: enabled},
		})
		require.NoError(t, database.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
			WorkspaceID: workspace.ID, DeploymentID: deployment.ID, RegionID: region,
			AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
			DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: time.Now().UnixMilli(),
		}))
		return deployment.ID
	}

	target := deploy("db", 5432, true)
	enrolled := []string{target, deploy("Bad_Slug", 8080, true)}
	var callerIDs []string
	for range callers {
		callerIDs = append(callerIDs, deploy("api", 8080, true))
	}
	savedBy := append([]string{deploy("api", 8080, false)}, callerIDs...)

	connectionID := seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID: workspace.ID, ProjectID: project.ID, CallerAppID: apps["api"], CallerEnvironmentID: environments["api"],
		TargetAppID: apps["db"], Name: "database",
		SelectionMode:      db.ConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: target, Valid: true},
	})
	savedConnectionIDs := []string{connectionID}
	for range 4 {
		savedConnectionIDs = append(savedConnectionIDs, uid.New(uid.ConnectionPrefix))
	}
	connections := make([]db.InsertDeploymentConnectionParams, 0, len(savedBy)*len(savedConnectionIDs))
	targets := make([]db.InsertDeploymentConnectionAppTargetParams, 0, len(savedBy)*len(savedConnectionIDs))
	for _, deployment := range savedBy {
		for i, savedConnectionID := range savedConnectionIDs {
			name := "database"
			if i > 0 {
				name = fmt.Sprintf("database-%d", i)
			}
			connections = append(connections, db.InsertDeploymentConnectionParams{
				DeploymentID: deployment, ConnectionID: savedConnectionID, WorkspaceID: workspace.ID, ProjectID: project.ID,
				AppID: apps["api"], EnvironmentID: environments["api"], ResourceType: "app", ResourceID: apps["db"],
				Name: name, CreatedAt: time.Now().UnixMilli(),
			})
			targets = append(targets, db.InsertDeploymentConnectionAppTargetParams{
				DeploymentID: deployment, ConnectionID: savedConnectionID,
				SelectionMode:      db.DeploymentConnectionAppTargetsSelectionModeDeployment,
				TargetDeploymentID: sql.NullString{String: target, Valid: true},
			})
		}
	}
	require.NoError(t, database.Bulk().InsertDeploymentConnections(ctx, connections))
	require.NoError(t, database.Bulk().InsertDeploymentConnectionAppTargets(ctx, targets))
	require.NoError(t, database.UpdateAppConnectionName(ctx, db.UpdateAppConnectionNameParams{
		ID: connectionID, Name: "changed-default", UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}))

	return privateNetworkFixture{
		callers: callerIDs, connectionIDs: savedConnectionIDs, replicas: append(enrolled, callerIDs...),
		target: target, regionID: region,
	}
}

func seedPrivateNetworkCluster(t *testing.T, seeder *seed.Seeder, name, platform string, cell sql.NullString) string {
	t.Helper()
	region := seeder.CreateRegion(t.Context(), seed.CreateRegionRequest{Name: name, Platform: platform})
	t.Cleanup(func() {
		require.NoError(t, seeder.DB.DeleteRegionsWithClusters(context.Background(), []string{region.ID}))
	})
	require.NoError(t, seeder.DB.UpsertCluster(t.Context(), db.UpsertClusterParams{
		ID: uid.New("cluster"), CellID: cell, RegionID: region.ID, LastHeartbeatAt: 0,
	}))
	return region.ID
}

func seedPrivateNetworkTopology(t *testing.T, seeder *seed.Seeder, platform string) (string, string) {
	t.Helper()
	secondCell := uid.New("cell")
	otherPlatform := strings.ToLower(uid.New("pf"))
	seedPrivateNetworkCluster(t, seeder, "second-"+platform, platform, sql.NullString{String: secondCell, Valid: true})
	seedPrivateNetworkCluster(t, seeder, "other-"+otherPlatform, otherPlatform, sql.NullString{String: uid.New("cell"), Valid: true})
	seedPrivateNetworkCluster(t, seeder, "null-"+platform, platform, sql.NullString{})
	return secondCell, "second-" + platform
}
