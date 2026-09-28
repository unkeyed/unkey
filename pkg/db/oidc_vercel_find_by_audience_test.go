package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestFindOIDCVercelIntegrationByAudience_LoadsSharedPermissions guarantees that
// Vercel OIDC integrations can use the same global permission rows as root keys.
func TestFindOIDCVercelIntegrationByAudience_LoadsSharedPermissions(t *testing.T) {
	mysqlConfig := containers.MySQL(t)
	database, err := New(Config{
		PrimaryDSN:  mysqlConfig.DSN,
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	ctx := t.Context()
	workspaceID := uid.New(uid.WorkspacePrefix)
	permissionWorkspaceID := uid.New(uid.WorkspacePrefix)
	integrationID := uid.New("vco")
	permissionID := uid.New(uid.PermissionPrefix)
	audience := "https://api.unkey.com/oidc/vercel/" + integrationID
	createdAtM := time.Now().UnixMilli()

	err = Query.InsertWorkspace(ctx, database.RW(), InsertWorkspaceParams{
		ID:           workspaceID,
		OrgID:        workspaceID,
		Name:         "OIDC customer workspace",
		Slug:         uid.New("slug"),
		CreatedAt:    createdAtM,
		K8sNamespace: uid.DNS1035(),
	})
	require.NoError(t, err)

	err = Query.InsertPermission(ctx, database.RW(), InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  permissionWorkspaceID,
		ProjectID:    uid.New(uid.ProjectPrefix),
		Name:         "api.*.read_api",
		Slug:         "api.*.read_api",
		Description:  dbtype.NullString{},
		CreatedAtM:   createdAtM,
	})
	require.NoError(t, err)

	_, err = database.RW().ExecContext(
		ctx,
		`INSERT INTO oidc_vercel
			(id, workspace_id, name, audience, vercel_team_slug, vercel_project_name, vercel_environment, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, true, ?)`,
		integrationID,
		workspaceID,
		"Production API",
		audience,
		"acme",
		"storefront",
		"production",
		createdAtM,
	)
	require.NoError(t, err)

	_, err = database.RW().ExecContext(
		ctx,
		`INSERT INTO oidc_vercel_permissions
			(oidc_vercel_id, permission_id, workspace_id, created_at)
		 VALUES (?, ?, ?, ?)`,
		integrationID,
		permissionID,
		workspaceID,
		createdAtM,
	)
	require.NoError(t, err)

	rows, err := Query.FindOIDCVercelIntegrationByAudience(ctx, database.RO(), audience)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].PermissionSlug.Valid)
	require.Equal(t, "api.*.read_api", rows[0].PermissionSlug.String)
}
