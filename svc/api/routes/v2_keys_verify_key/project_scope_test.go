package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestVerifyKeyUsesKeyspaceProjectForURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	keyspaceProjectID := api.ProjectID
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	apiProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "API project",
		Slug:        uid.New("project"),
	})
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE apis SET project_id = ? WHERE id = ?", apiProject.ID, api.ID)
	require.NoError(t, err)

	call := func(t *testing.T, projectID string) openapi.V2KeysVerifyKeyResponseData {
		t.Helper()
		permission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#verify", workspace.ID, projectID, api.KeyAuthID.String, key.KeyID)
		rootKey := h.CreateRootKey(workspace.ID, permission)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, handler.Request{Key: key.Key})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		return res.Body.Data
	}

	t.Run("API project does not authorize the keyspace", func(t *testing.T) {
		data := call(t, apiProject.ID)
		require.Equal(t, openapi.NOTFOUND, data.Code)
		require.False(t, data.Valid)
	})

	t.Run("keyspace project authorizes the keyspace", func(t *testing.T) {
		data := call(t, keyspaceProjectID)
		require.Equal(t, openapi.VALID, data.Code)
		require.True(t, data.Valid)
	})
}

// TestVerifyKeyIgnoresDirectPermissionsFromAnotherProject guarantees that a
// malformed association cannot grant a permission across project boundaries.
func TestVerifyKeyIgnoresDirectPermissionsFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other permission project",
		Slug:        uid.New("project"),
	})
	permissionID := uid.New(uid.PermissionPrefix)
	permissionSlug := "cross-project.direct"
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		ProjectID:    otherProject.ID,
		Name:         permissionSlug,
		Slug:         permissionSlug,
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertKeyPermission(t.Context(), h.DB.RW(), db.InsertKeyPermissionParams{
		KeyID:        key.KeyID,
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		CreatedAt:    time.Now().UnixMilli(),
	}))

	verifyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#verify",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, verifyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Key: key.Key, Permissions: ptr.P(permissionSlug)})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.INSUFFICIENTPERMISSIONS, res.Body.Data.Code)
	require.False(t, res.Body.Data.Valid)
	require.Empty(t, res.Body.Data.Permissions)
}

// TestVerifyKeyIgnoresRolePermissionsFromAnotherProject guarantees that a
// malformed role association cannot import a permission from another project.
func TestVerifyKeyIgnoresRolePermissionsFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: "local-role"})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other role permission project",
		Slug:        uid.New("project"),
	})
	permissionID := uid.New(uid.PermissionPrefix)
	permissionSlug := "cross-project.role-permission"
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		ProjectID:    otherProject.ID,
		Name:         permissionSlug,
		Slug:         permissionSlug,
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertRolePermission(t.Context(), h.DB.RW(), db.InsertRolePermissionParams{
		RoleID:       role.ID,
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertKeyRole(t.Context(), h.DB.RW(), db.InsertKeyRoleParams{
		KeyID:       key.KeyID,
		RoleID:      role.ID,
		WorkspaceID: workspace.ID,
		CreatedAtM:  time.Now().UnixMilli(),
	}))

	verifyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#verify",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, verifyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Key: key.Key, Permissions: ptr.P(permissionSlug)})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.INSUFFICIENTPERMISSIONS, res.Body.Data.Code)
	require.False(t, res.Body.Data.Valid)
	require.Equal(t, []string{role.Name}, res.Body.Data.Roles)
	require.Empty(t, res.Body.Data.Permissions)
}

// TestVerifyKeyIgnoresRolesFromAnotherProject guarantees that a malformed key
// association cannot import a role and its permissions from another project.
func TestVerifyKeyIgnoresRolesFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other role project",
		Slug:        uid.New("project"),
	})
	roleID := uid.New(uid.RolePrefix)
	roleName := "cross-project-role"
	require.NoError(t, db.Query.InsertRole(t.Context(), h.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		ProjectID:   otherProject.ID,
		Name:        roleName,
		Description: sql.NullString{Valid: false},
		CreatedAt:   time.Now().UnixMilli(),
	}))
	permissionID := uid.New(uid.PermissionPrefix)
	permissionSlug := "cross-project.role"
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		ProjectID:    api.ProjectID,
		Name:         permissionSlug,
		Slug:         permissionSlug,
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertRolePermission(t.Context(), h.DB.RW(), db.InsertRolePermissionParams{
		RoleID:       roleID,
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertKeyRole(t.Context(), h.DB.RW(), db.InsertKeyRoleParams{
		KeyID:       key.KeyID,
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		CreatedAtM:  time.Now().UnixMilli(),
	}))

	verifyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#verify",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, verifyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Key: key.Key, Permissions: ptr.P(permissionSlug)})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.INSUFFICIENTPERMISSIONS, res.Body.Data.Code)
	require.False(t, res.Body.Data.Valid)
	require.Empty(t, res.Body.Data.Roles)
	require.Empty(t, res.Body.Data.Permissions)
}

// TestVerifyKeyIgnoresIdentitiesFromAnotherProject guarantees that a malformed
// key association cannot expose identity data across project boundaries.
func TestVerifyKeyIgnoresIdentitiesFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other identity project",
		Slug:        uid.New("project"),
	})
	identityID := uid.New(uid.IdentityPrefix)
	require.NoError(t, db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  "cross-project-identity",
		WorkspaceID: workspace.ID,
		ProjectID:   otherProject.ID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte(`{"private":"value"}`),
	}))
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identityID),
	})

	verifyPermission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#verify",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
		key.KeyID,
	)
	rootKey := h.CreateRootKey(workspace.ID, verifyPermission)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Key: key.Key})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.VALID, res.Body.Data.Code)
	require.True(t, res.Body.Data.Valid)
	require.Nil(t, res.Body.Data.Identity)
}
