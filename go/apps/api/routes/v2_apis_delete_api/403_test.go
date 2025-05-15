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

func TestAuthorizationErrors(t *testing.T) {
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

	// Create an API for testing

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

	// Test case for insufficient permissions - missing delete_api
	t.Run("missing delete_api permission", func(t *testing.T) {
		// Create a root key with only read_api but no delete_api permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	// Test case for permission for different API
	t.Run("permission for different API", func(t *testing.T) {
		// Create a root key with permissions for a specific different API
		differentApiId := "api_different"
		rootKey := h.CreateRootKey(
			workspace.ID,
			fmt.Sprintf("api.%s.delete_api", differentApiId),
		)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID, // Using the test API, not the one we have permission for
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace

		// Create a root key for the other workspace
		rootKey := h.CreateRootKey(uid.New(uid.TestPrefix), "api.*.delete_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID, // API is in the original workspace
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Note: We mask wrong workspace errors as 404 not found
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "The provided root key is invalid. The requested workspace does not exist.", res.Body.Error.Detail)
	})
}
