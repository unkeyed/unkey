package handler_test

import (
	"context"
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_create_key"
)

func TestCreateKeyUsesKeyspaceProjectForURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Vault: h.Vault}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	keyspaceProjectID := api.ProjectID
	apiProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "API project",
		Slug:        uid.New("project"),
	})
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE apis SET project_id = ? WHERE id = ?", apiProject.ID, api.ID)
	require.NoError(t, err)

	request := handler.Request{ApiId: api.ID}
	call := func(t *testing.T, permission string) int {
		t.Helper()
		rootKey := h.CreateRootKey(workspace.ID, permission)
		res := testutil.CallRoute[handler.Request, openapi.V2KeysCreateKeyResponseBody](h, route, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, request)
		return res.Status
	}

	t.Run("API project does not authorize the keyspace", func(t *testing.T) {
		require.Equal(t, http.StatusNotFound, call(t, createKeyPermission(workspace.ID, apiProject.ID, api.KeyAuthID.String)))
	})

	t.Run("keyspace project authorizes the keyspace", func(t *testing.T) {
		require.Equal(t, http.StatusOK, call(t, createKeyPermission(workspace.ID, keyspaceProjectID, api.KeyAuthID.String)))
	})
}

func TestCreateKeyRejectsPermissionFromAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Vault: h.Vault}
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
	require.NotEqual(t, otherProjectAPI.ProjectID, keyProjectAPI.ProjectID)

	permissionSlug := "documents.read.create.wrong-project"
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

	rootKey := h.CreateRootKey(workspace.ID, createKeyPermission(workspace.ID, keyProject.ID, keyProjectAPI.KeyAuthID.String))
	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		ApiId:       keyProjectAPI.ID,
		Permissions: ptr.P([]string{permissionSlug}),
	})

	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, permissionSlug)

	permissionsInKeyProject, err := db.Query.FindPermissionsBySlugs(t.Context(), h.DB.RO(), db.FindPermissionsBySlugsParams{
		WorkspaceID: workspace.ID,
		ProjectID:   keyProject.ID,
		Slugs:       []string{permissionSlug},
	})
	require.NoError(t, err)
	require.Empty(t, permissionsInKeyProject)
}
