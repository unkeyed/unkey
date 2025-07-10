package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestSuccess(t *testing.T) {
	// ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace
	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")

	api := h.CreateApi(workspace.ID, false)

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("verifies key as valid", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
		})

		req := handler.Request{
			ApiId: api.ID,
			Key:   key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Code, openapi.VALID, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)
	})

	t.Run("verifies expired key as valid and then invalid", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Expires:     ptr.P(time.Now().Add(time.Second * 3)),
		})

		req := handler.Request{
			ApiId: api.ID,
			Key:   key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Code, openapi.VALID, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		time.Sleep(time.Second * 3)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Code, openapi.EXPIRED, "Key should be expired but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	t.Run("disabled key", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Disabled:    true,
		})

		req := handler.Request{
			ApiId: api.ID,
			Key:   key.Key,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Code, openapi.DISABLED, "Key should be disabled but got %s", res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be invalid but got %t", res.Body.Data.Valid)
	})

	// returns metadata returns identity
	// ratelimit check
	// ratelimit override
	// with cost

}
