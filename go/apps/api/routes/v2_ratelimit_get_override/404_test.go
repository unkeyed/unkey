package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestOverrideNotFound(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace but no override
	namespaceID := uid.New("test_ns")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        uid.New("test"),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.read_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with non-existent identifier
	t.Run("override not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: "non_existent_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_override_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
	})

	// Test with non-existent namespace
	t.Run("namespace not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  "ns_nonexistent",
			Identifier: "some_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	// Test with non-existent namespace name
	t.Run("namespace name not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  "nonexistent_namespace",
			Identifier: "some_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	// Test with non-existent identifier
	t.Run("identifier not found", func(t *testing.T) {

		namespaceID := uid.New(uid.TestPrefix)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.read_override", namespaceID))
		err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
			ID:          namespaceID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        uid.New(uid.TestPrefix),
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		nonExistentIdentifier := "nonexistent_identifier"
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: nonExistentIdentifier,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{fmt.Sprintf("Bearer %s", rootKey)},
		}, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_override_not_found", res.Body.Error.Type)
	})
}
