package handler_test

import (
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_permissions"
)

// TestAddPermissionsRejectsPermissionFromAnotherProject guarantees that a key
// cannot receive a permission from a different project.
func TestAddPermissionsRejectsPermissionFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	otherProjectAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	keyProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Key project",
		Slug:        uid.New("project"),
	})
	keyProjectAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID, ProjectID: keyProject.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyProjectAPI.KeyAuthID.String,
		Name:        ptr.P("project-scoped-key"),
	})

	permissionSlug := "documents.read.add.wrong-project"
	err := db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: uid.New(uid.PermissionPrefix),
		WorkspaceID:  workspace.ID,
		ProjectID:    otherProjectAPI.ProjectID,
		Name:         permissionSlug,
		Slug:         permissionSlug,
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	writeKey := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#write", workspace.ID, keyProject.ID, keyProjectAPI.KeyAuthID.String, key.KeyID)
	writePermission := fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/*#write", workspace.ID, keyProject.ID)
	rootKey := h.CreateRootKey(workspace.ID, writeKey, writePermission)
	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		KeyId:       key.KeyID,
		Permissions: []string{permissionSlug},
	})

	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, permissionSlug)

	permissions, err := db.Query.ListDirectPermissionsByKeyID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Empty(t, permissions)
}
