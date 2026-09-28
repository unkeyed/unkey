package cluster

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestPrivateNetworkSnapshotStreamsEveryPageThenCompletes guarantees that the
// private network snapshot has no size limit: Ctrl reads every page, streams
// all apps in chunks, and ends with a complete chunk whose total matches, so
// Krane can tell a whole snapshot from a truncated one. It also guarantees the
// snapshot is authenticated and scoped to the caller's platform.
func TestPrivateNetworkSnapshotStreamsEveryPageThenCompletes(t *testing.T) {
	server := containers.MySQL(t)
	database, err := db.New(server.DSN, sqlcomment.Static{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	platform := strings.ToLower(uid.New("pf"))
	cell := uid.New("cell")
	callers, target := seedPrivateNetwork(t, database, platform, cell)

	clusterCache, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 1, Resource: "test_private_network_clusters", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(clusterCache.Close)
	_, handler := ctrlv1connect.NewClusterServiceHandler(&Service{db: database, bearer: "test-bearer", clusterCache: clusterCache})
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	client := ctrlv1connect.NewClusterServiceClient(httpServer.Client(), httpServer.URL)

	receive := func(bearer string) ([]*ctrlv1.PrivateNetworkStateChunk, error) {
		request := connect.NewRequest(&ctrlv1.StreamPrivateNetworkStateRequest{
			Cluster: &ctrlv1.ClusterKey{CellId: cell, Region: "region-" + platform, Platform: platform},
		})
		request.Header().Set("Authorization", "Bearer "+bearer)
		stream, err := client.StreamPrivateNetworkState(t.Context(), request)
		if err != nil {
			return nil, err
		}
		defer func() { require.NoError(t, stream.Close()) }()
		var chunks []*ctrlv1.PrivateNetworkStateChunk
		for stream.Receive() {
			chunks = append(chunks, stream.Msg())
		}
		return chunks, stream.Err()
	}

	_, err = receive("wrong-bearer")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	chunks, err := receive("test-bearer")
	require.NoError(t, err)
	require.Len(t, chunks, 4, "two pages of bindings and replicas stream as three app chunks and one complete chunk")
	last := chunks[len(chunks)-1]
	require.True(t, last.GetComplete())
	require.Empty(t, last.GetApps())

	bindings, replicas := map[string]*ctrlv1.PrivateNetworkApp{}, map[string]*ctrlv1.PrivateNetworkApp{}
	for _, chunk := range chunks[:len(chunks)-1] {
		require.False(t, chunk.GetComplete())
		require.LessOrEqual(t, len(chunk.GetApps()), privateNetworkPageSize)
		for _, app := range chunk.GetApps() {
			if strings.HasPrefix(app.GetBindingId(), "self-") {
				require.NotContains(t, replicas, app.GetCallerDeploymentId())
				replicas[app.GetCallerDeploymentId()] = app
				continue
			}
			require.NotContains(t, bindings, app.GetCallerDeploymentId())
			bindings[app.GetCallerDeploymentId()] = app
		}
	}
	require.Equal(t, uint64(len(bindings)+len(replicas)), last.GetTotal())
	require.Len(t, bindings, callers, "every caller deployment across both binding pages")
	require.Len(t, replicas, callers+1, "every caller and the target publish replicas; the app with an invalid slug does not")
	for caller, app := range bindings {
		require.Equal(t, "database", app.GetBindingName(), "binding for %s", caller)
		require.Equal(t, target, app.GetDeploymentId(), "binding for %s", caller)
		require.Equal(t, int32(5432), app.GetPort(), "binding for %s", caller)
	}
	for deployment, app := range replicas {
		require.Equal(t, deployment, app.GetDeploymentId())
		require.Equal(t, "self-"+deployment, app.GetBindingId())
		require.Contains(t, []string{"api", "db"}, app.GetBindingName(), "replicas resolve under their app slug")
	}
}

func seedPrivateNetwork(t *testing.T, database db.Database, platform, cell string) (int, string) {
	t.Helper()
	const callers = privateNetworkPageSize + 1
	workspace, project, region, target := uid.New("ws"), uid.New("proj"), uid.New("reg"), uid.New("dep")
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := database.RW().ExecContext(t.Context(), query, args...)
		require.NoError(t, err)
	}
	// The seed is committed so the RPC's own transaction sees it. t.Context is
	// already canceled when cleanup runs.
	t.Cleanup(func() {
		for _, statement := range []string{
			`DELETE FROM app_bindings WHERE workspace_id = ?`,
			`DELETE FROM deployment_topology WHERE workspace_id = ?`,
			`DELETE FROM deployments WHERE workspace_id = ?`,
			`DELETE FROM environments WHERE workspace_id = ?`,
			`DELETE FROM apps WHERE workspace_id = ?`,
			`DELETE FROM projects WHERE workspace_id = ?`,
			`DELETE FROM workspaces WHERE id = ?`,
		} {
			_, err := database.RW().ExecContext(context.Background(), statement, workspace)
			require.NoError(t, err)
		}
		_, err := database.RW().ExecContext(context.Background(), `DELETE FROM clusters WHERE region_id = ?`, region)
		require.NoError(t, err)
		_, err = database.RW().ExecContext(context.Background(), `DELETE FROM regions WHERE id = ?`, region)
		require.NoError(t, err)
	})
	exec(`INSERT INTO regions (id,name,platform) VALUES (?,?,?)`, region, "region-"+platform, platform)
	exec(`INSERT INTO clusters (id,cell_id,region_id,last_heartbeat_at) VALUES (?,?,?,0)`, uid.New("cluster"), cell, region)
	exec(`INSERT INTO workspaces (id,org_id,name,slug,k8s_namespace,beta_features) VALUES (?,?,'Workspace',?,?,'{}')`,
		workspace, uid.New("org"), strings.ToLower(workspace), strings.ReplaceAll(strings.ToLower(workspace), "_", "-"))
	exec(`INSERT INTO projects (id,workspace_id,name,slug,created_at) VALUES (?,?,'Project','project',1)`, project, workspace)
	for _, app := range []string{"api", "db", "Bad_Slug"} {
		exec(`INSERT INTO apps (id,workspace_id,project_id,name,slug,source_type,created_at) VALUES (?,?,?,?,?,'git',1)`,
			workspace+"-"+app, workspace, project, app, app)
		exec(`INSERT INTO environments (id,workspace_id,project_id,app_id,slug,kind,created_at) VALUES (?,?,?,?,'preview','preview',1)`,
			workspace+"-"+app+"-env", workspace, project, workspace+"-"+app)
	}
	insertDeployment := func(id, app string, port int) {
		t.Helper()
		exec(`INSERT INTO deployments (id,k8s_name,workspace_id,project_id,environment_id,app_id,sentinel_config,cpu_millicores,memory_mib,desired_state,encrypted_environment_variables,status,port,created_at)
			VALUES (?,?,?,?,?,?,'{}',100,128,'running','{}','ready',?,1)`, id, id, workspace, project, workspace+"-"+app+"-env", workspace+"-"+app, port)
		exec(`INSERT INTO deployment_topology (workspace_id,deployment_id,region_id,desired_status,created_at) VALUES (?,?,?,'running',1)`, workspace, id, region)
	}
	insertDeployment(target, "db", 5432)
	insertDeployment(uid.New("dep"), "Bad_Slug", 8080)
	exec(`INSERT INTO deployments (id,k8s_name,workspace_id,project_id,environment_id,app_id,sentinel_config,cpu_millicores,memory_mib,desired_state,encrypted_environment_variables,status,port,created_at)
		SELECT CONCAT(?, n), CONCAT(?, n), ?, ?, ?, ?, '{}', 100, 128, 'running', '{}', 'ready', 8080, 1
		FROM (SELECT a.n + 10 * b.n + 100 * c.n + 1000 * d.n AS n
			FROM (SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) a,
				(SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) b,
				(SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) c,
				(SELECT 0 n UNION ALL SELECT 1) d) numbers
		WHERE n < ?`,
		workspace+"-caller-", workspace+"-caller-", workspace, project, workspace+"-api-env", workspace+"-api", callers)
	exec(`INSERT INTO deployment_topology (workspace_id,deployment_id,region_id,desired_status,created_at)
		SELECT workspace_id, id, ?, 'running', 1 FROM deployments WHERE app_id = ?`, region, workspace+"-api")
	exec(`INSERT INTO app_bindings (id,workspace_id,project_id,app_id,environment_id,resource_type,resource_id,name,selection_mode,target_deployment_id,created_at)
		VALUES (?,?,?,?,?,'app',?,'database','deployment',?,1)`,
		uid.New("binding"), workspace, project, workspace+"-api", workspace+"-api-env", workspace+"-db", target)
	return callers, target
}
