package db_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	oidcdb "github.com/unkeyed/unkey/pkg/auth/oidc/vercel/db"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestFindOIDCVercelIntegrationByAudience_IsolatesPrincipalPermissions guarantees
// that grants from another workspace, principal type, or principal are not loaded.
func TestFindOIDCVercelIntegrationByAudience_IsolatesPrincipalPermissions(t *testing.T) {
	mysqlConfig := containers.MySQL(t)
	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlConfig.DSN,
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	ctx := t.Context()
	workspaceID := uid.New(uid.WorkspacePrefix)
	integrationID := uid.New("vco")
	audience := "https://api.unkey.com/oidc/vercel/" + integrationID
	createdAtM := time.Now().UnixMilli()

	err = db.Query.InsertWorkspace(ctx, database.RW(), db.InsertWorkspaceParams{
		ID:           workspaceID,
		OrgID:        workspaceID,
		Name:         "OIDC customer workspace",
		Slug:         uid.New("slug"),
		CreatedAt:    createdAtM,
		K8sNamespace: uid.DNS1035(),
	})
	require.NoError(t, err)

	err = db.Query.InsertOIDCVercelIntegration(ctx, database.RW(), db.InsertOIDCVercelIntegrationParams{
		ID:                integrationID,
		WorkspaceID:       workspaceID,
		Name:              "Production API",
		Audience:          audience,
		VercelTeamSlug:    "acme",
		VercelProjectName: "storefront",
		VercelEnvironment: "production",
		Enabled:           true,
		CreatedAt:         createdAtM,
	})
	require.NoError(t, err)

	for _, grant := range []db.InsertUnkeyPermissionParams{
		{WorkspaceID: uid.New(uid.WorkspacePrefix), PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeOidc, PrincipalID: integrationID, Slug: "api.*.delete_api"},
		{WorkspaceID: workspaceID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: integrationID, Slug: "api.*.create_api"},
		{WorkspaceID: workspaceID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeOidc, PrincipalID: uid.New("vco"), Slug: "api.*.create_key"},
	} {
		grant.ID = uid.New(uid.PermissionPrefix)
		grant.CreatedAt = createdAtM
		require.NoError(t, db.Query.InsertUnkeyPermission(ctx, database.RW(), grant))
	}

	rows, err := oidcdb.Query.FindOIDCVercelIntegrationByAudience(ctx, database.RO(), audience)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, integrationID, rows[0].ID)
	require.False(t, rows[0].PermissionSlug.Valid)

	require.NoError(t, db.Query.InsertUnkeyPermission(ctx, database.RW(), db.InsertUnkeyPermissionParams{
		ID:            uid.New(uid.PermissionPrefix),
		WorkspaceID:   workspaceID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeOidc,
		PrincipalID:   integrationID,
		Slug:          "api.*.read_api",
		CreatedAt:     createdAtM,
	}))

	rows, err = oidcdb.Query.FindOIDCVercelIntegrationByAudience(ctx, database.RO(), audience)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].PermissionSlug.Valid)
	require.Equal(t, "api.*.read_api", rows[0].PermissionSlug.String)
}
