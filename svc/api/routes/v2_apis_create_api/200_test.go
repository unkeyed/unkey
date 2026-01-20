package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
)

// TestCreateApiSuccessfully verifies the complete API creation workflow including
// authentication, authorization, database operations, and various API name formats.
// This comprehensive test covers the happy path scenarios and edge cases for valid
// API creation requests.
func TestCreateApiSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test creating an API manually via DB to verify queries
	// This test validates that the underlying database queries work correctly
	// by bypassing the HTTP handler and directly testing the DB operations.
	t.Run("insert api via DB", func(t *testing.T) {
		createdAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), createdAPI.ID)
		require.NoError(t, err)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
		require.True(t, api.KeyAuthID.Valid)
		require.Equal(t, createdAPI.KeyAuthID.String, api.KeyAuthID.String)
	})

	// Test creating a basic API
	// This test verifies the complete end-to-end flow through the HTTP handler,
	// including authentication, transaction handling, and response validation.
	// It confirms that the API, keyring, and audit logs are created correctly.
	t.Run("create basic api", func(t *testing.T) {
		apiName := "test-api-basic"
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.ApiId)

		// Verify the API in the database
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
		require.True(t, api.AuthType.Valid)
		require.Equal(t, db.ApisAuthTypeKey, api.AuthType.ApisAuthType)
		require.True(t, api.KeyAuthID.Valid)
		require.NotEmpty(t, api.KeyAuthID.String)
		require.False(t, api.DeletedAtM.Valid)

		// Verify the key space was created
		keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), api.KeyAuthID.String)
		require.NoError(t, err)
		require.Equal(t, h.Resources().UserWorkspace.ID, keySpace.WorkspaceID)
		require.False(t, keySpace.DeletedAtM.Valid)

		// Verify the audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(auditLogs), 1)

		var foundCreateEvent bool
		for _, log := range auditLogs {
			if log.AuditLog.Event == "api.create" {
				foundCreateEvent = true
				require.Equal(t, h.Resources().UserWorkspace.ID, log.AuditLog.WorkspaceID)
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find an api.create audit log event")
	})

	// Test creating multiple APIs
	// This test ensures that multiple API creation requests work correctly
	// and that each API gets a unique identifier, preventing ID collisions.
	t.Run("create multiple apis", func(t *testing.T) {
		apiNames := []string{"api-1", "api-2", "api-3"}
		apiIds := make([]string, len(apiNames))

		for i, name := range apiNames {
			req := handler.Request{
				Name: name,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotEmpty(t, res.Body.Data.ApiId)

			apiIds[i] = res.Body.Data.ApiId
		}

		// Verify all APIs were created with unique IDs
		apiIdSet := make(map[string]bool)
		for _, apiId := range apiIds {
			apiIdSet[apiId] = true
		}
		require.Equal(t, len(apiNames), len(apiIdSet), "All API IDs should be unique")

		// Verify each API in the database
		for i, apiId := range apiIds {
			api, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiId)
			require.NoError(t, err)
			require.Equal(t, apiNames[i], api.Name)
		}
	})

	// Test with a longer API name
	// This test validates that API names with many characters are handled
	// correctly and stored properly in the database without truncation.
	t.Run("create api with long name", func(t *testing.T) {
		apiName := "my-super-awesome-production-api-for-customer-management-and-analytics"
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
	})

	// This test verifies that UUID-style names are accepted and that delete
	// protection is properly set to false by default for new APIs.
	t.Run("create api with UUID name", func(t *testing.T) {
		req := handler.Request{
			Name: uid.New(uid.TestPrefix),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.ApiId)

		// Verify the API in the database
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, req.Name, api.Name)

		// Verify delete protection is false (specifically tested in TypeScript)
		require.True(t, api.DeleteProtection.Valid)
		require.False(t, api.DeleteProtection.Bool)
	})

}
