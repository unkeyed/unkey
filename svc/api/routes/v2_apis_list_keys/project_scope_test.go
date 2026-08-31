package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

func TestListKeysUsesKeyspaceProjectForURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Vault: h.Vault, ApiCache: h.Caches.LiveApiByID}
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
	h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})

	permissionsFor := func(projectID string) []string {
		return []string{
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read_keyspace", workspace.ID, projectID, api.KeyAuthID.String),
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#read_key", workspace.ID, projectID, api.KeyAuthID.String),
		}
	}
	headersFor := func(permissions ...string) http.Header {
		rootKey := h.CreateRootKey(workspace.ID, permissions...)
		return http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
	}

	t.Run("API project does not authorize the keyspace", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headersFor(permissionsFor(apiProject.ID)...), handler.Request{ApiId: api.ID})
		require.Equal(t, http.StatusNotFound, res.Status)
	})

	t.Run("keyspace project authorizes the keyspace", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(permissionsFor(keyspaceProjectID)...), handler.Request{ApiId: api.ID})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, 1)
	})
}
