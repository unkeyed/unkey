package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// A small connection table and multiple workspace IDs allow a full-scan plan.
// Cleanup must finish while another workspace's connection is still uncommitted.
func TestWorkspaceCleanupDoesNotLockOtherWorkspaceConnections(t *testing.T) {
	database, err := sql.Open("mysql", containers.MySQLIsolated(t).DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	queries := NewQueries(database)
	workspace := uid.New(uid.WorkspacePrefix)
	workspaceSecond := uid.New(uid.WorkspacePrefix)
	workspaceOther := uid.New(uid.WorkspacePrefix)
	project := uid.New(uid.ProjectPrefix)
	app := uid.New(uid.AppPrefix)
	environment := uid.New(uid.EnvironmentPrefix)
	insertTestWorkspace(t, queries, workspace, uid.DNS1035())
	insertTestWorkspace(t, queries, workspaceSecond, uid.DNS1035())
	insertTestWorkspace(t, queries, workspaceOther, uid.DNS1035())
	insertTestProject(t, queries, workspaceOther, project)
	insertTestApp(t, queries, workspaceOther, project, app, "caller")
	insertTestEnvironment(t, queries, workspaceOther, project, app, environment, "preview", mysqltype.EnvironmentKindPreview)

	tx, err := database.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tx.Rollback()) })
	require.NoError(t, NewQueries(tx).InsertAppConnection(t.Context(), InsertAppConnectionParams{
		ID: uid.New(uid.ConnectionPrefix), WorkspaceID: workspaceOther, ProjectID: project,
		AppID: app, EnvironmentID: environment, ResourceType: AppConnectionsResourceTypeApp,
		ResourceID: uid.New(uid.AppPrefix), Name: "target", CreatedAt: time.Now().UnixMilli(),
	}))

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(t, queries.DeleteWorkspacesWithChildren(ctx, []string{workspace, workspaceSecond}))
	_, err = queries.FindWorkspaceByID(t.Context(), workspace)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = queries.FindWorkspaceByID(t.Context(), workspaceSecond)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = queries.FindWorkspaceByID(t.Context(), workspaceOther)
	require.NoError(t, err)
}
