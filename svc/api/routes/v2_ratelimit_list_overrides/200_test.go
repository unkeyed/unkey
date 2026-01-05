package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_list_overrides"
)

func TestListOverridesSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New("test_ns")
	namespaceName := "test_namespace"
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create an override
	identifier := "test_identifier"
	limit := int32(10)
	duration := int32(1000)
	overrideID := uid.New(uid.RatelimitOverridePrefix)

	err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    duration,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:     h.DB,
		Keys:   h.Keys,
		Logger: h.Logger,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.read_override")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test getting by namespace name
	t.Run("get by namespace name", func(t *testing.T) {
		req := handler.Request{
			Namespace: namespaceName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.RawBody)
		require.NotNil(t, res.Body)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, overrideID, res.Body.Data[0].OverrideId)
		require.Equal(t, identifier, res.Body.Data[0].Identifier)
		require.Equal(t, int64(limit), res.Body.Data[0].Limit)
		require.Equal(t, int64(duration), res.Body.Data[0].Duration)
	})

	// Test getting by namespace ID
	t.Run("get by namespace ID", func(t *testing.T) {
		req := handler.Request{
			Namespace: namespaceID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, overrideID, res.Body.Data[0].OverrideId)
		require.Equal(t, identifier, res.Body.Data[0].Identifier)
		require.Equal(t, int64(limit), res.Body.Data[0].Limit)
		require.Equal(t, int64(duration), res.Body.Data[0].Duration)
	})

	// Test getting empty list when no overrides exist
	t.Run("get empty list for namespace without overrides", func(t *testing.T) {
		// Create a namespace without any overrides
		emptyNamespaceID := uid.New("empty_ns")
		emptyNamespaceName := "empty_namespace"
		err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
			ID:          emptyNamespaceID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        emptyNamespaceName,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			Namespace: emptyNamespaceID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.RawBody)
		require.NotNil(t, res.Body)
		require.Empty(t, res.Body.Data, "expected empty array when no overrides exist")
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})
}
