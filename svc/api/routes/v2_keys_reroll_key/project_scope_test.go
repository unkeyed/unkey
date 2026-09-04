package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

// TestRerollKeyDoesNotCopyPermissionsFromAnotherProject guarantees that key
// rotation does not propagate a malformed cross-project association.
func TestRerollKeyDoesNotCopyPermissionsFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other reroll project",
		Slug:        uid.New("project"),
	})
	permissionID := uid.New(uid.PermissionPrefix)
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		ProjectID:    otherProject.ID,
		Name:         "cross-project.reroll",
		Slug:         "cross-project.reroll",
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertKeyPermission(t.Context(), h.DB.RW(), db.InsertKeyPermissionParams{
		KeyID:        key.KeyID,
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		CreatedAt:    time.Now().UnixMilli(),
	}))

	writeKeyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#write",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, writeKeyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{KeyId: key.KeyID, Expiration: 0})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	permissions, err := db.Query.ListDirectPermissionsByKeyID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Empty(t, permissions)
}

// TestRerollKeyDoesNotCopyRolesFromAnotherProject guarantees that key rotation
// does not propagate a malformed cross-project role association.
func TestRerollKeyDoesNotCopyRolesFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other reroll role project",
		Slug:        uid.New("project"),
	})
	roleID := uid.New(uid.RolePrefix)
	require.NoError(t, db.Query.InsertRole(t.Context(), h.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		ProjectID:   otherProject.ID,
		Name:        "cross-project-reroll-role",
		Description: sql.NullString{Valid: false},
		CreatedAt:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertKeyRole(t.Context(), h.DB.RW(), db.InsertKeyRoleParams{
		KeyID:       key.KeyID,
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		CreatedAtM:  time.Now().UnixMilli(),
	}))

	writeKeyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#write",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, writeKeyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{KeyId: key.KeyID, Expiration: 0})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	rerolled, err := db.Query.FindLiveKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	keyData := db.ToKeyData(rerolled)
	require.NotNil(t, keyData)
	require.Empty(t, keyData.Roles)
}

// TestRerollKeyDoesNotCopyIdentitiesFromAnotherProject guarantees that key
// rotation does not propagate a malformed cross-project identity association.
func TestRerollKeyDoesNotCopyIdentitiesFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other reroll identity project",
		Slug:        uid.New("project"),
	})
	identityID := uid.New(uid.IdentityPrefix)
	require.NoError(t, db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  "cross-project-reroll-identity",
		WorkspaceID: workspace.ID,
		ProjectID:   otherProject.ID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	}))
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  &identityID,
	})

	writeKeyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#write",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, writeKeyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{KeyId: key.KeyID, Expiration: 0})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	rerolled, err := db.Query.FindLiveKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.False(t, rerolled.KeyIdentityID.Valid)
}
