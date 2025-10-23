package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_delete_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteOverrideSuccessfully(t *testing.T) {
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
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       10,
		Duration:    1000,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Auditlogs:               h.Auditlogs,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.delete_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test deleting by namespace name
	t.Run("delete by namespace name", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: identifier,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)

		// Verify the override was deleted (check soft delete)
		var override db.RatelimitOverride
		override, err = db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			OverrideID:  overrideID,
		})

		require.NoError(t, err)
		require.True(t, override.DeletedAtM.Valid, "Override should be marked as deleted")
	})

	// Test deleting by namespace ID
	t.Run("delete by namespace ID", func(t *testing.T) {
		// Create another override to test ID-based deletion
		identifier2 := "test_identifier_2"
		overrideID2 := uid.New(uid.RatelimitOverridePrefix)
		err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID2,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: namespaceID,
			Identifier:  identifier2,
			Limit:       10,
			Duration:    1000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: identifier2,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)

		// Verify the override was deleted (check soft delete)
		override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			OverrideID:  overrideID2,
		})

		require.NoError(t, err)
		require.True(t, override.DeletedAtM.Valid, "Override should be marked as deleted")
	})
}
