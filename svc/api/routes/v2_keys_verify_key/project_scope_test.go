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
