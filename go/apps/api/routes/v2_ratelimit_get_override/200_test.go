package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestGetOverrideSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New("test_ns")
	namespaceName := "test_namespace"
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources.UserWorkspace.ID,
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
		WorkspaceID: h.Resources.UserWorkspace.ID,
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    duration,
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

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.read_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test getting by namespace name
	t.Run("get by namespace name", func(t *testing.T) {
		req := handler.Request{
			NamespaceName: &namespaceName,
			Identifier:    identifier,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.Equal(t, overrideID, res.Body.OverrideId)
		require.Equal(t, namespaceID, res.Body.NamespaceId)
		require.Equal(t, identifier, res.Body.Identifier)
		require.Equal(t, int64(limit), res.Body.Limit)
		require.Equal(t, int64(duration), res.Body.Duration)
	})

	// Test getting by namespace ID
	t.Run("get by namespace ID", func(t *testing.T) {
		req := handler.Request{
			NamespaceId: &namespaceID,
			Identifier:  identifier,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.Equal(t, overrideID, res.Body.OverrideId)
		require.Equal(t, namespaceID, res.Body.NamespaceId)
		require.Equal(t, identifier, res.Body.Identifier)
		require.Equal(t, int64(limit), res.Body.Limit)
		require.Equal(t, int64(duration), res.Body.Duration)
	})
}
