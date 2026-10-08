package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type countingTx struct {
	db.DBTX
	statements int
}

func (c *countingTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	c.statements++
	return c.DBTX.ExecContext(ctx, query, args...)
}

func (c *countingTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	c.statements++
	return c.DBTX.QueryContext(ctx, query, args...)
}

func (c *countingTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	c.statements++
	return c.DBTX.QueryRowContext(ctx, query, args...)
}

func TestInsertConnectionsBatchesStatements(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "project", Slug: slug(uid.ProjectPrefix),
	})
	newApp := func() (string, string) {
		t.Helper()
		app := seeder.CreateApp(ctx, seed.CreateAppRequest{
			ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "app", Slug: slug(uid.AppPrefix),
		})
		environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
			Slug: "production", Kind: mysqltype.EnvironmentKindProduction,
		})
		return app.ID, environment.ID
	}
	callerAppID, callerEnvironmentID := newApp()
	targetAppID, targetEnvironmentID := newApp()
	var pinned []string
	for range 3 {
		pinned = append(pinned, seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			WorkspaceID: workspaceID, ProjectID: project.ID, AppID: targetAppID, EnvironmentID: targetEnvironmentID,
			Status: mysqltype.DeploymentsStatusReady,
		}).ID)
	}

	for _, test := range []struct{ connections, statements int }{
		{0, 0}, {3, 3}, {300, 3}, {1000, 3}, {1001, 5},
	} {
		t.Run(fmt.Sprintf("%d connections", test.connections), func(t *testing.T) {
			payload := deployPayload{
				Target: db.FindDeployTargetRow{
					WorkspaceID: workspaceID, ProjectID: project.ID, AppID: callerAppID, EnvironmentID: callerEnvironmentID,
				},
				CreatedAt: time.Now().UnixMilli(),
			}
			for i := range test.connections {
				connection := db.ListAppConnectionsByAppRow{
					ID: uid.New(uid.ConnectionPrefix), Name: fmt.Sprintf("target-%d", i), ResourceType: "app", ResourceID: targetAppID,
				}
				switch i % 3 {
				case 0:
					connection.SelectionMode = db.NullConnectionAppTargetsSelectionMode{ConnectionAppTargetsSelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic, Valid: true}
				case 1:
					connection.SelectionMode = db.NullConnectionAppTargetsSelectionMode{ConnectionAppTargetsSelectionMode: db.ConnectionAppTargetsSelectionModeEnvironment, Valid: true}
					connection.TargetEnvironmentID = sql.NullString{String: targetEnvironmentID, Valid: true}
				case 2:
					connection.SelectionMode = db.NullConnectionAppTargetsSelectionMode{ConnectionAppTargetsSelectionMode: db.ConnectionAppTargetsSelectionModeDeployment, Valid: true}
					connection.TargetDeploymentID = sql.NullString{String: pinned[(i/3)%len(pinned)], Valid: true}
				}
				payload.Connections = append(payload.Connections, connection)
			}

			deploymentID := uid.New(uid.DeploymentPrefix)
			counting := &countingTx{DBTX: nil, statements: 0}
			require.NoError(t, db.Tx(ctx, database.RW(), func(txCtx context.Context, tx db.DBTX) error {
				counting.DBTX = tx
				return insertConnections(txCtx, counting, deploymentID, payload)
			}))
			require.Equal(t, test.statements, counting.statements)

			saved, err := database.ListDeploymentConnectionsByDeploymentId(ctx, deploymentID)
			require.NoError(t, err)
			require.Len(t, saved, test.connections)
		})
	}
}

func slug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
