package testutil_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func TestHarnessDatabaseOwnershipAndIsolation(t *testing.T) {
	shared := testutil.NewHarness(t)
	workspace := shared.Resources().UserWorkspace
	var closedDB db.Database

	t.Run("isolated", func(t *testing.T) {
		isolated := testutil.NewHarness(t, testutil.HarnessConfig{}, testutil.HarnessConfig{IsolatedMySQL: true})
		closedDB = isolated.DB
		_, err := db.Query.FindWorkspaceByID(t.Context(), isolated.DB.RO(), workspace.ID)
		require.ErrorIs(t, err, sql.ErrNoRows)

		require.NoError(t, db.Query.InsertWorkspace(t.Context(), isolated.DB.RW(), db.InsertWorkspaceParams{
			ID:           workspace.ID,
			OrgID:        workspace.OrgID,
			Name:         "isolated workspace",
			Slug:         workspace.Slug,
			CreatedAt:    workspace.CreatedAtM,
			K8sNamespace: workspace.K8sNamespace,
		}))
		isolatedWorkspace, err := db.Query.FindWorkspaceByID(t.Context(), isolated.DB.RO(), workspace.ID)
		require.NoError(t, err)
		require.Equal(t, "isolated workspace", isolatedWorkspace.Name)
	})

	_, err := db.Query.FindWorkspaceByID(t.Context(), closedDB.RO(), workspace.ID)
	require.ErrorContains(t, err, "database is closed")

	peer := testutil.NewHarness(t)
	persisted, err := db.Query.FindWorkspaceByID(t.Context(), peer.DB.RO(), workspace.ID)
	require.NoError(t, err)
	require.Equal(t, workspace.Name, persisted.Name)

	t.Run("shared clients close without deleting suite rows", func(t *testing.T) {
		child := testutil.NewHarness(t)
		closedDB = child.DB
	})

	_, err = db.Query.FindWorkspaceByID(t.Context(), closedDB.RO(), workspace.ID)
	require.ErrorContains(t, err, "database is closed")

	persisted, err = db.Query.FindWorkspaceByID(t.Context(), shared.DB.RO(), workspace.ID)
	require.NoError(t, err)
	require.Equal(t, workspace.Name, persisted.Name)
}

func TestEnvironmentTopology(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.CreateWorkspace()
	production := h.CreateEnvironmentTopology(workspace.ID, mysqltype.EnvironmentKindProduction)
	repeated := h.CreateEnvironmentTopology(workspace.ID, mysqltype.EnvironmentKindProduction)
	preview := h.CreateEnvironmentTopology(h.Resources().UserWorkspace.ID, mysqltype.EnvironmentKindPreview)
	require.Equal(t, workspace.ID, production.WorkspaceID)
	require.Equal(t, workspace.ID, repeated.WorkspaceID)
	require.Equal(t, h.Resources().UserWorkspace.ID, preview.WorkspaceID)
	require.NotEqual(t, production.ProjectID, repeated.ProjectID)
	require.NotEqual(t, production.AppID, repeated.AppID)
	require.NotEqual(t, production.EnvironmentID, repeated.EnvironmentID)

	for _, fixture := range []struct {
		workspaceID   string
		projectID     string
		appID         string
		environmentID string
		kind          mysqltype.EnvironmentKind
	}{
		{
			workspace.ID,
			production.ProjectID,
			production.AppID,
			production.EnvironmentID,
			mysqltype.EnvironmentKindProduction,
		},
		{
			workspace.ID,
			repeated.ProjectID,
			repeated.AppID,
			repeated.EnvironmentID,
			mysqltype.EnvironmentKindProduction,
		},
		{
			h.Resources().UserWorkspace.ID,
			preview.ProjectID,
			preview.AppID,
			preview.EnvironmentID,
			mysqltype.EnvironmentKindPreview,
		},
	} {
		project, err := db.Query.FindProjectById(t.Context(), h.DB.RO(), fixture.projectID)
		require.NoError(t, err)
		require.Equal(t, fixture.workspaceID, project.WorkspaceID)

		app, err := db.Query.FindAppById(t.Context(), h.DB.RO(), fixture.appID)
		require.NoError(t, err)
		require.Equal(t, fixture.workspaceID, app.WorkspaceID)
		require.Equal(t, fixture.projectID, app.ProjectID)

		environment, err := db.Query.FindEnvironmentById(t.Context(), h.DB.RO(), fixture.environmentID)
		require.NoError(t, err)
		require.Equal(t, fixture.workspaceID, environment.WorkspaceID)
		require.Equal(t, fixture.projectID, environment.ProjectID)
		require.Equal(t, fixture.appID, environment.AppID)
		require.Equal(t, fixture.kind, environment.Kind)

		_, err = db.Query.FindAppBuildSettingByAppEnv(t.Context(), h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID:         fixture.appID,
			EnvironmentID: fixture.environmentID,
		})
		require.NoError(t, err)

		_, err = db.Query.FindAppRuntimeSettingsByAppAndEnv(t.Context(), h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID:         fixture.appID,
			EnvironmentID: fixture.environmentID,
		})
		require.NoError(t, err)
	}

	_, err := db.Query.FindWorkspaceBillingByWorkspaceID(t.Context(), h.DB.RO(), workspace.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
