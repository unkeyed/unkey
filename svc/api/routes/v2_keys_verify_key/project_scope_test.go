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
	apiProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "API project",
		Slug:        uid.New("project"),
	})
	keyspaceProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Keyspace project",
		Slug:        uid.New("project"),
	})
	keyspaceProjectID := keyspaceProject.ID
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(context.Background(), h.DB.RW(), db.InsertKeySpaceParams{
		ID:          keySpaceID,
		WorkspaceID: workspace.ID,
		ProjectID:   keyspaceProjectID,
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)
	apiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(context.Background(), h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api",
		WorkspaceID: workspace.ID,
		ProjectID:   apiProject.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)
	key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: keySpaceID})

	call := func(t *testing.T, projectID string) openapi.V2KeysVerifyKeyResponseData {
		t.Helper()
		permission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#verify", workspace.ID, projectID, keySpaceID, key.KeyID)
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
