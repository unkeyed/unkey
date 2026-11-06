package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	deleteHandler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_delete_key"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		AuditLogBuffer:   h.AuditLogBuffer,
		KeyVerifications: h.KeyVerifications,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("key not found", func(t *testing.T) {
		req := handler.Request{
			Key: uid.New("test"),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "Key should be not found but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	// nolint:exhaustruct
	bareNotFound := openapi.V2KeysVerifyKeyResponseData{Code: openapi.NOTFOUND, Valid: false}
	verify := func(t *testing.T, req handler.Request) openapi.V2KeysVerifyKeyResponseData {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		return res.Body.Data
	}

	t.Run("soft deleted key", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Deleted:     true,
		})

		// The second call answers from the cached tombstone.
		require.Equal(t, bareNotFound, verify(t, handler.Request{Key: key.Key}))
		require.Equal(t, bareNotFound, verify(t, handler.Request{Key: key.Key}))
	})

	t.Run("soft deleted key in another workspace", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		otherApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspace.ID})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: otherWorkspace.ID,
			KeySpaceID:  otherApi.KeyAuthID.String,
			Deleted:     true,
		})

		require.Equal(t, bareNotFound, verify(t, handler.Request{Key: key.Key}))
	})

	t.Run("key deleted through the API after a cached verification", func(t *testing.T) {
		deleteRoute := &deleteHandler.Handler{
			DB:        h.DB,
			Auditlogs: h.Auditlogs,
			KeyCache:  h.Caches.VerificationKeyByHash,
		}
		h.Register(deleteRoute)
		deleteRootKey := h.CreateRootKey(workspace.ID, "api.*.delete_key")
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

		require.Equal(t, openapi.VALID, verify(t, handler.Request{Key: key.Key}).Code)

		res := testutil.CallRoute[deleteHandler.Request, deleteHandler.Response](h, deleteRoute, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", deleteRootKey)},
		}, deleteHandler.Request{KeyId: key.KeyID})
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)

		require.Equal(t, bareNotFound, verify(t, handler.Request{Key: key.Key}))
	})

	t.Run("key belongs to different workspace", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		otherApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspace.ID})
		otherKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: otherWorkspace.ID,
			KeySpaceID:  otherApi.KeyAuthID.String,
		})

		req := handler.Request{
			Key: otherKey.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "Key should be not found but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})
}
