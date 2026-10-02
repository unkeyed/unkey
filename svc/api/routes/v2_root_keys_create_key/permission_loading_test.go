package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestLegacyRootKeyIgnoresNewPermissions guarantees legacy authentication only
// loads legacy direct and role assignments, even when the new store has a grant
// for the same key ID.
func TestLegacyRootKeyIgnoresNewPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:    r.RootWorkspace.ID,
		KeySpaceID:     r.RootKeySpace.ID,
		ForWorkspaceID: &r.UserWorkspace.ID,
		Permissions: []seed.CreatePermissionRequest{{
			WorkspaceID: r.RootWorkspace.ID,
			Name:        "api.*.read_key",
			Slug:        "api.*.read_key",
		}},
	})
	require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), db.InsertUnkeyPermissionParams{
		ID:            uid.New(uid.PermissionPrefix),
		WorkspaceID:   r.UserWorkspace.ID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   key.KeyID,
		Slug:          "unkey:v1:" + r.UserWorkspace.ID + ":rootKeys/*#write",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+key.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	for range 2 {
		root, err := h.Keys.GetRootKey(t.Context(), session)
		require.NoError(t, err)
		require.Equal(t, []string{"api.*.read_key"}, root.Permissions)
	}
}

// TestNewRootKeyPermissionLoadingIsScoped guarantees new authentication only
// loads matching new-store grants. Legacy assignments, other workspaces, OIDC
// principals, and other root keys cannot supply permissions.
func TestNewRootKeyPermissionLoadingIsScoped(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	newPermission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	legacyPermission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "legacy",
		Slug:        "legacy.permission",
	})
	key := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: []string{newPermission},
	})
	for _, row := range []struct {
		workspaceID   string
		principalType db.UnkeyPrincipalPermissionsPrincipalType
		principalID   string
		slug          string
	}{
		{h.CreateWorkspace().ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, key.KeyID, "wrong-workspace"},
		{workspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeOidc, key.KeyID, "wrong-type"},
		{workspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, uid.New(uid.KeyPrefix), "wrong-principal"},
	} {
		require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), db.InsertUnkeyPermissionParams{
			ID:            uid.New(uid.PermissionPrefix),
			WorkspaceID:   row.workspaceID,
			PrincipalType: row.principalType,
			PrincipalID:   row.principalID,
			Slug:          row.slug,
			CreatedAt:     h.Clock.Now().UnixMilli(),
		}))
	}
	require.NoError(t, db.Query.InsertKeyPermission(t.Context(), h.DB.RW(), db.InsertKeyPermissionParams{
		KeyID:        key.KeyID,
		PermissionID: legacyPermission.ID,
		WorkspaceID:  workspace.ID,
		CreatedAt:    h.Clock.Now().UnixMilli(),
		UpdatedAt:    sql.NullInt64{},
	}))

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+key.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	for range 2 {
		root, err := h.Keys.GetRootKey(t.Context(), session)
		require.NoError(t, err)
		require.Equal(t, []string{newPermission}, root.Permissions)
	}
}

// TestUnkeyPermissionUniquenessIncludesPrincipalScope guarantees the same
// permission can belong to two keys or principal types, but cannot occur twice
// for one principal in one workspace. A different workspace remains independent.
func TestUnkeyPermissionUniquenessIncludesPrincipalScope(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	principalID := uid.New(uid.KeyPrefix)
	permission := "unkey:v1:" + workspaceID + ":rootKeys/*#write"
	row := db.InsertUnkeyPermissionParams{
		ID:            uid.New(uid.PermissionPrefix),
		WorkspaceID:   workspaceID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   principalID,
		Slug:          permission,
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}
	require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), row))
	row.ID = uid.New(uid.PermissionPrefix)
	err := db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), row)
	require.True(t, db.IsDuplicateKeyError(err), "duplicate permission must be rejected: %v", err)
	row.PrincipalID = uid.New(uid.KeyPrefix)
	require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), row))
	row.ID = uid.New(uid.PermissionPrefix)
	row.PrincipalID = principalID
	row.PrincipalType = db.UnkeyPrincipalPermissionsPrincipalTypeOidc
	require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), row))
	row.ID = uid.New(uid.PermissionPrefix)
	row.PrincipalType = db.UnkeyPrincipalPermissionsPrincipalTypeRootKey
	row.WorkspaceID = h.CreateWorkspace().ID
	require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), row))
}

// TestUnkeyPermissionsRejectUnknownPrincipalTypes guarantees unsupported principal
// types cannot enter storage. For example, "unknown" is rejected instead of
// creating a permission that no supported principal can use.
func TestUnkeyPermissionsRejectUnknownPrincipalTypes(t *testing.T) {
	h := testutil.NewHarness(t)
	err := db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), db.InsertUnkeyPermissionParams{
		ID:            uid.New(uid.PermissionPrefix),
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalType("unknown"),
		PrincipalID:   uid.New(uid.KeyPrefix),
		Slug:          "unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	})
	require.Error(t, err)
}
