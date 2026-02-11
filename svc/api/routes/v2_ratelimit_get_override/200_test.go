package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_get_override"
)

func TestGetOverrideSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New("test_ns")
	namespaceName := uid.New("test")
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
		DB:                      h.DB,
		Keys:                    h.Keys,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.read_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test getting by namespace name
	t.Run("get by namespace name", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: identifier,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", *res.Body)
		require.NotNil(t, res.Body)
		require.Equal(t, overrideID, res.Body.Data.OverrideId)
		require.Equal(t, identifier, res.Body.Data.Identifier)
		require.Equal(t, int64(limit), res.Body.Data.Limit)
		require.Equal(t, int64(duration), res.Body.Data.Duration)
	})

	// Test getting by namespace ID
	t.Run("get by namespace ID", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: identifier,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.Equal(t, overrideID, res.Body.Data.OverrideId)
		require.Equal(t, identifier, res.Body.Data.Identifier)
		require.Equal(t, int64(limit), res.Body.Data.Limit)
		require.Equal(t, int64(duration), res.Body.Data.Duration)
	})
}
