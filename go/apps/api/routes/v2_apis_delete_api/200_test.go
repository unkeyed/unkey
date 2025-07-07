package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}

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
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: false, String: ""},
			DefaultBytes:       sql.NullInt32{Valid: false, Int32: 0},
			StoreEncryptedKeys: false,
		})
		require.NoError(t, err)

		apiID := uid.New(uid.APIPrefix)
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        "Test API",
			WorkspaceID: h.Resources().UserWorkspace.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)

		require.Equal(t, apiID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Delete the API
		req := handler.Request{
			ApiId: apiID,
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
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err) // Should still find it, just marked as deleted
		require.True(t, apiAfterDelete.DeletedAtM.Valid)
	})

	// Test case for deleting an API with active keys
	t.Run("delete api with active keys", func(t *testing.T) {
		// Create keyring for the API
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: false, String: ""},
			DefaultBytes:       sql.NullInt32{Valid: false, Int32: 0},
			StoreEncryptedKeys: false,
		})
		require.NoError(t, err)

		// Create the API
		apiID := uid.New(uid.APIPrefix)
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        "Test API With Keys",
			WorkspaceID: h.Resources().UserWorkspace.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a key associated with this API
		keyID := uid.New(uid.KeyPrefix)
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeyringID:   keyAuthID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
			// Add other required fields based on your schema
			Hash:  hash.Sha256(uid.New(uid.TestPrefix)),
			Start: "teststart",
		})
		require.NoError(t, err)

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.Equal(t, apiID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Delete the API
		req := handler.Request{
			ApiId: apiID,
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
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)

		// Check that the key is still accessible (soft delete doesn't cascade to keys)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Equal(t, keyID, key.ID)
	})

	// Test case for deleting an API immediately after creation
	t.Run("delete api immediately after creation", func(t *testing.T) {
		// Create keyring for the API
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: false, String: ""},
			DefaultBytes:       sql.NullInt32{Valid: false, Int32: 0},
			StoreEncryptedKeys: false,
		})
		require.NoError(t, err)

		// Create the API
		apiID := uid.New(uid.APIPrefix)
		apiName := "Test Immediate Delete API"
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify the API was created
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.Equal(t, apiID, apiBeforeDelete.ID)
		require.Equal(t, apiName, apiBeforeDelete.Name)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Immediately delete the API without any delay
		req := handler.Request{
			ApiId: apiID,
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
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)
	})
}
