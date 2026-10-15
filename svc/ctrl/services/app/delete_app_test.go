package app

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeleteAppMarksBeforeWorkerRuns(t *testing.T) {
	for _, status := range []int{http.StatusAccepted, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			ctx := t.Context()
			database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })
			seeder := seed.New(t, database, nil)
			seeder.Seed(ctx)
			workspaceID := seeder.Resources.UserWorkspace.ID
			project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
				ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "Delete", Slug: "delete",
			})
			app := seeder.CreateApp(ctx, seed.CreateAppRequest{ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "App", Slug: "app"})
			sibling := seeder.CreateApp(ctx, seed.CreateAppRequest{ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "Sibling", Slug: "sibling"})
			before := time.Now().Add(time.Hour).UnixMilli()
			for _, appID := range []string{app.ID, sibling.ID} {
				for range 2 {
					environmentID := uid.New(uid.EnvironmentPrefix)
					for range 2 {
						deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{ID: uid.New(uid.DeploymentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: appID, EnvironmentID: environmentID, Status: "pending"})
						for range 2 {
							_, err := database.RW().ExecContext(ctx, `INSERT INTO deployment_topology
								(workspace_id, deployment_id, region_id, desired_status, created_at, updated_at)
								VALUES (?, ?, ?, 'running', ?, ?)`, workspaceID, deployment.ID, uid.New("region"), before, before)
							require.NoError(t, err)
						}
					}
				}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				if _, err := io.WriteString(w, `{"invocationId":"inv_test","status":"Accepted"}`); err != nil {
					t.Errorf("write ingress response: %v", err)
				}
			}))
			t.Cleanup(server.Close)
			svc := New(Config{Database: database, Restate: restateingress.NewClient(server.URL), Bearer: "test-token"})
			req := connect.NewRequest(&ctrlv1.DeleteAppRequest{AppId: app.ID, Actor: &ctrlv1.ActorInfo{}})
			req.Header().Set("Authorization", "Bearer test-token")
			_, err = svc.DeleteApp(ctx, req)
			var deletedAt sql.NullInt64
			require.NoError(t, database.RW().QueryRowContext(ctx, "SELECT deleted_at FROM apps WHERE id = ?", app.ID).Scan(&deletedAt))
			if status == http.StatusAccepted {
				require.NoError(t, err)
				require.True(t, deletedAt.Valid)
				_, err = svc.DeleteApp(ctx, req)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			} else {
				require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
				require.False(t, deletedAt.Valid)
			}
			for _, appID := range []string{app.ID, sibling.ID} {
				var minUpdatedAt, maxUpdatedAt int64
				require.NoError(t, database.RW().QueryRowContext(ctx, `SELECT MIN(dt.updated_at), MAX(dt.updated_at)
					FROM deployment_topology dt JOIN deployments d ON d.id = dt.deployment_id WHERE d.app_id = ?`, appID).Scan(&minUpdatedAt, &maxUpdatedAt))
				if status == http.StatusAccepted && appID == app.ID {
					require.Greater(t, minUpdatedAt, before)
				} else {
					require.Equal(t, before, minUpdatedAt)
					require.Equal(t, before, maxUpdatedAt)
				}
			}
			_, err = database.RW().ExecContext(ctx, "UPDATE projects SET deleted_at = ? WHERE id = ?", before, project.ID)
			require.NoError(t, err)
			req.Msg.AppId = sibling.ID
			_, err = svc.DeleteApp(ctx, req)
			require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		})
	}
}
