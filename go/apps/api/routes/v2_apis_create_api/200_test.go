package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_create_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateApiSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test creating an API manually via DB to verify queries
	t.Run("insert api via DB", func(t *testing.T) {
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:            keyAuthID,
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			CreatedAtM:    time.Now().UnixMilli(),
			DefaultPrefix: sql.NullString{Valid: false, String: ""},
			DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
		})
		require.NoError(t, err)

		apiID := uid.New(uid.APIPrefix)
		apiName := "test-api-db"
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		api, err := db.Query.FindApiById(ctx, h.DB.RO(), apiID)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
		require.True(t, api.KeyAuthID.Valid)
		require.Equal(t, keyAuthID, api.KeyAuthID.String)
	})

	// Test creating a basic API
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
		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
		require.True(t, api.AuthType.Valid)
		require.Equal(t, db.ApisAuthTypeKey, api.AuthType.ApisAuthType)
		require.True(t, api.KeyAuthID.Valid)
		require.NotEmpty(t, api.KeyAuthID.String)
		require.False(t, api.DeletedAtM.Valid)

		// Verify the key auth was created
		keyAuth, err := db.Query.FindKeyringByID(ctx, h.DB.RO(), api.KeyAuthID.String)
		require.NoError(t, err)
		require.Equal(t, h.Resources().UserWorkspace.ID, keyAuth.WorkspaceID)
		require.False(t, keyAuth.DeletedAtM.Valid)

		// Verify the audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetById(ctx, h.DB.RO(), res.Body.Data.ApiId)
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
			api, err := db.Query.FindApiById(ctx, h.DB.RO(), apiId)
			require.NoError(t, err)
			require.Equal(t, apiNames[i], api.Name)
		}
	})

	// Test with a longer API name
	t.Run("create api with long name", func(t *testing.T) {
		apiName := "my-super-awesome-production-api-for-customer-management-and-analytics"
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
	})

	// Test with special characters in name
	t.Run("create api with special characters", func(t *testing.T) {
		apiName := "special_api-123!@#$%^&*()"
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
	})

	t.Run("create api with UUID name", func(t *testing.T) {
		apiName := uid.New("uuid-test-") // Using uid.New to generate a unique ID
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.ApiId)

		// Verify the API in the database
		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)

		// Verify delete protection is false (specifically tested in TypeScript)
		require.True(t, api.DeleteProtection.Valid)
		require.False(t, api.DeleteProtection.Bool)
	})

	// Test with minimum name length (exactly 3 characters)
	t.Run("create api with minimum length name", func(t *testing.T) {
		apiName := "min" // Exactly 3 characters
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.ApiId)

		// Verify the API in the database
		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
	})

	// Test with name containing only numeric characters
	t.Run("create api with numeric name", func(t *testing.T) {
		apiName := "12345" // Only numeric characters
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.ApiId)

		// Verify the API in the database
		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
	})

	// Test with name containing Unicode characters
	t.Run("create api with unicode name", func(t *testing.T) {
		apiName := "测试-api-🔑" // Unicode characters including emoji
		req := handler.Request{
			Name: apiName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.ApiId)

		// Verify the API in the database
		api, err := db.Query.FindApiById(ctx, h.DB.RO(), res.Body.Data.ApiId)
		require.NoError(t, err)
		require.Equal(t, apiName, api.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
	})
}
