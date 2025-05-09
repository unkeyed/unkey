package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
		Caches:      h.Caches,
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.delete_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for deleting an API without keys
	t.Run("delete api without keys", func(t *testing.T) {
		// Create an API for testing
		api, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			Name:        "Test API Without Keys",
			WorkspaceID: workspace.ID,
		})
		require.NoError(t, err)

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiById(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Delete the API
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiById(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err) // Should still find it, just marked as deleted
		require.True(t, apiAfterDelete.DeletedAtM.Valid)

		// Verify audit log was created
		auditLogs, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "auditlogs" WHERE "event" = 'api.delete' AND "resourceId" = $1`,
			api.ID)
		require.NoError(t, err)
		require.True(t, auditLogs.Next(), "Audit log for API deletion should exist")
		auditLogs.Close()
	})

	// Test case for deleting an API with keys
	t.Run("delete api with keys", func(t *testing.T) {
		// Create an API with keys
		api, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			Name:        "Test API With Keys",
			WorkspaceID: workspace.ID,
		})
		require.NoError(t, err)

		// Create a KeyAuth for the API
		keyAuth, err := db.Query.InsertKeyAuth(ctx, h.DB.RW(), db.InsertKeyAuthParams{
			ApiID: api.ID,
		})
		require.NoError(t, err)

		// Update the API with KeyAuthID
		_, err = db.Query.UpdateApiSetKeyAuthId(ctx, h.DB.RW(), api.ID, keyAuth.ID)
		require.NoError(t, err)

		// Create test keys
		testKeys := []db.InsertKeyParams{
			{
				Start:       "key1_",
				KeyAuthID:   keyAuth.ID,
				WorkspaceID: workspace.ID,
				Name:        db.NewNullString("Test Key 1"),
			},
			{
				Start:       "key2_",
				KeyAuthID:   keyAuth.ID,
				WorkspaceID: workspace.ID,
				Name:        db.NewNullString("Test Key 2"),
			},
		}

		var keyIds []string
		for _, params := range testKeys {
			key, err := db.Query.InsertKey(ctx, h.DB.RW(), params)
			require.NoError(t, err)
			keyIds = append(keyIds, key.ID)
		}

		// Delete the API
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiById(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)

		// Verify all keys are marked as deleted
		for _, keyId := range keyIds {
			key, err := db.Query.FindKeyById(ctx, h.DB.RO(), keyId)
			require.NoError(t, err)
			require.True(t, key.DeletedAtM.Valid, "Key should be marked as deleted")
		}

		// Verify audit logs were created for API and keys
		apiAuditLogs, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "auditlogs" WHERE "event" = 'api.delete' AND "resourceId" = $1`,
			api.ID)
		require.NoError(t, err)
		require.True(t, apiAuditLogs.Next(), "Audit log for API deletion should exist")
		apiAuditLogs.Close()

		for _, keyId := range keyIds {
			keyAuditLogs, err := h.DB.RO().QueryContext(ctx,
				`SELECT * FROM "auditlogs" WHERE "event" = 'key.delete' AND "resourceId" = $1`,
				keyId)
			require.NoError(t, err)
			require.True(t, keyAuditLogs.Next(), "Audit log for key deletion should exist")
			keyAuditLogs.Close()
		}
	})
}
