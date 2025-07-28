package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
	})

	t.Run("root key with wrong api permissions", func(t *testing.T) {
		api2 := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api2.KeyAuthID.String,
		})
		rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.verify_key", api.ID))

		req := handler.Request{
			Key: key.Key,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "Key should be not found but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("root key without sufficient permissions", func(t *testing.T) {
		// Create root key with insufficient permissions
		limitedRootKey := h.CreateRootKey(workspace.ID, "api.*.read") // Wrong permission

		req := handler.Request{
			Key: key.Key,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", limitedRootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "Key should be not found but got %s", res.Body.Data.Code)
	})
}
