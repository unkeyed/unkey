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
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSpecialCharactersInApiId(t *testing.T) {
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

	// Test case for API ID with special characters
	t.Run("api id with special characters", func(t *testing.T) {
		// Test with API IDs containing special characters
		// Note: Keep these within what your uid.New function and database allow
		specialApiIDs := []string{
			"api_with-hyphen",
			"api_with.dot",
			"api_with_underscore",
			"api_123456",
			"api_with-multiple.special_chars-123",
		}

		for _, apiID := range specialApiIDs {
			t.Run(apiID, func(t *testing.T) {
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

				err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
					ID:          apiID,
					Name:        fmt.Sprintf("Test API %s", apiID),
					WorkspaceID: h.Resources().UserWorkspace.ID,
					AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
					KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
					CreatedAtM:  time.Now().UnixMilli(),
				})
				require.NoError(t, err)

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

				require.Equal(t, 200, res.Status, "Failed to delete API with ID: %s", apiID)

				// Verify API is marked as deleted
				apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
				require.NoError(t, err)
				require.True(t, apiAfterDelete.DeletedAtM.Valid)
			})
		}
	})
}
