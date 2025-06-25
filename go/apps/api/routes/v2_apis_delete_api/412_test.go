package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteProtection(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
		Caches:      h.Caches,
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

	// Test case for deleting an API with delete protection enabled
	t.Run("delete protected API", func(t *testing.T) {

		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			CreatedAtM:         h.Clock.Now().UnixMilli(),
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

		err = db.Query.UpdateApiDeleteProtection(ctx, h.DB.RW(), db.UpdateApiDeleteProtectionParams{
			ApiID:            apiID,
			DeleteProtection: sql.NullBool{Valid: true, Bool: true},
		})
		require.NoError(t, err)

		// Ensure API exists and has delete protection
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.Equal(t, apiID, apiBeforeDelete.ID)
		require.True(t, apiBeforeDelete.DeleteProtection.Valid)
		require.True(t, apiBeforeDelete.DeleteProtection.Bool)

		// Attempt to delete the API
		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "This API has delete protection enabled. Disable it before attempting to delete.", res.Body.Error.Detail)

		// Verify API was NOT deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.False(t, apiAfterDelete.DeletedAtM.Valid, "API should not have been deleted")
	})
}
