package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_delete_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFound(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace but no override
	namespaceID := uid.New("test_ns")
	namespaceName := "test_namespace"
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources.UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.delete_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with non-existent identifier
	t.Run("override not found", func(t *testing.T) {
		req := handler.Request{
			NamespaceId: &namespaceID,
			Identifier:  "non_existent_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundError](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/not_found", res.Body.Type)
		require.Equal(t, http.StatusNotFound, res.Body.Status)
	})

	// Test with non-existent namespace
	t.Run("namespace not found", func(t *testing.T) {
		nonExistentNamespaceId := "ns_nonexistent"
		req := handler.Request{
			NamespaceId: &nonExistentNamespaceId,
			Identifier:  "some_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundError](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/not_found", res.Body.Type)
	})

	// Test with non-existent namespace name
	t.Run("namespace name not found", func(t *testing.T) {
		nonExistentNamespaceName := "nonexistent_namespace"
		req := handler.Request{
			NamespaceName: &nonExistentNamespaceName,
			Identifier:    "some_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundError](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/not_found", res.Body.Type)
	})
}
