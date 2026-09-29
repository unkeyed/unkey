package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestUnkeyPermissionLoadingIsScoped guarantees only the matching root key loads
// its direct permissions, on both database and cache reads. For example, another
// workspace, an OIDC principal, and an ordinary API key cannot supply permissions
// to that root key. Existing legacy permissions remain available during migration.
func TestUnkeyPermissionLoadingIsScoped(t *testing.T) {
	for _, isRootKey := range []bool{true, false} {
		name := "ordinary key"
		if isRootKey {
			name = "root key"
		}
		t.Run(name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			r := h.Resources()
			var forWorkspaceID *string
			if isRootKey {
				forWorkspaceID = &r.UserWorkspace.ID
			}
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:    r.RootWorkspace.ID,
				KeySpaceID:     r.RootKeySpace.ID,
				ForWorkspaceID: forWorkspaceID,
				Permissions: []seed.CreatePermissionRequest{{
					WorkspaceID: r.RootWorkspace.ID,
					Name:        "api.*.read_key",
					Slug:        "api.*.read_key",
				}},
			})
			permission := "unkey:v1:" + r.UserWorkspace.ID + ":rootKeys/*#write"
			for _, row := range []struct {
				workspaceID   string
				principalType db.UnkeyPrincipalPermissionsPrincipalType
				principalID   string
				slug          string
			}{
				{r.UserWorkspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, key.KeyID, permission},
				{h.CreateWorkspace().ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, key.KeyID, "wrong-workspace"},
				{r.UserWorkspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeOidc, key.KeyID, "wrong-type"},
				{r.UserWorkspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, uid.New(uid.KeyPrefix), "wrong-principal"},
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
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.Header.Set("Authorization", "Bearer "+key.Key)
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
			for range 2 {
				loaded, err := h.Keys.Get(t.Context(), nil, hash.Sha256(key.Key))
				require.NoError(t, err)
				require.Equal(t, []string{"api.*.read_key"}, loaded.Permissions)
				root, err := h.Keys.GetRootKey(t.Context(), session)
				if isRootKey {
					require.NoError(t, err)
					require.ElementsMatch(t, []string{"api.*.read_key", permission}, root.Permissions)
				} else {
					require.Error(t, err)
				}
			}
		})
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
