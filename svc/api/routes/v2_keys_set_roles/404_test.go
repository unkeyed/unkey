package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_set_roles"
)

func TestNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create test data using testutil helper
	defaultPrefix := "test"
	defaultBytes := int32(16)
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		DefaultPrefix: &defaultPrefix,
		DefaultBytes:  &defaultBytes,
	})

	keyName := "Valid Test Key"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
	})
	validKeyID := keyResponse.KeyID

	// Create a valid role
	validRoleID := uid.New(uid.TestPrefix)
	validRoleName := "valid-test-role"
	err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		RoleID:      validRoleID,
		WorkspaceID: workspace.ID,
		Name:        validRoleName,
		Description: sql.NullString{Valid: true, String: "Valid test role"},
	})
	require.NoError(t, err)

	// Test case for non-existent key ID
	t.Run("non-existent key ID", func(t *testing.T) {
		nonExistentKeyID := "key_nonexistent123456789"

		req := handler.Request{
			KeyId: nonExistentKeyID,
			Roles: []string{validRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
		require.Equal(t, 404, res.Body.Error.Status)
	})

	// Test case for non-existent role ID
	t.Run("non-existent role ID", func(t *testing.T) {
		nonExistentRoleID := "role_nonexistent123456789"

		req := handler.Request{
			KeyId: validKeyID,
			Roles: []string{nonExistentRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Role '%s' was not found", nonExistentRoleID))
		require.Equal(t, 404, res.Body.Error.Status)
	})

	// Test case for non-existent role name
	t.Run("non-existent role name", func(t *testing.T) {
		nonExistentRoleName := "nonexistent-role-name"

		req := handler.Request{
			KeyId: validKeyID,
			Roles: []string{nonExistentRoleName},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Role '%s' was not found", nonExistentRoleName))
		require.Equal(t, 404, res.Body.Error.Status)
	})

	// Test case for key from different workspace (workspace isolation)
	t.Run("key from different workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspaceID := uid.New("test_ws")
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      uid.New("test_name"),
			Slug:      uid.New("slug"),
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a keyring and key in the other workspace
		otherKeySpaceID := uid.New(uid.KeySpacePrefix)
		err = db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
			ID:                 otherKeySpaceID,
			WorkspaceID:        otherWorkspaceID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "other"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		otherKeyID := uid.New(uid.KeyPrefix)
		otherKeyString := "other_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                otherKeyID,
			KeySpaceID:        otherKeySpaceID,
			Hash:              hash.Sha256(otherKeyString),
			Start:             otherKeyString[:4],
			WorkspaceID:       otherWorkspaceID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Other Workspace Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: otherKeyID,
			Roles: []string{validRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
		require.Equal(t, 404, res.Body.Error.Status)
	})

	// Test case for role from different workspace (workspace isolation)
	t.Run("role from different workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspaceID := uid.New("test_ws")
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      uid.New("test_name"),
			Slug:      uid.New("slug"),
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a role in the other workspace
		otherRoleID := uid.New(uid.TestPrefix)
		otherRoleName := "other-workspace-role"
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      otherRoleID,
			WorkspaceID: otherWorkspaceID,
			Name:        otherRoleName,
			Description: sql.NullString{Valid: true, String: "Role in other workspace"},
		})
		require.NoError(t, err)

		// Test with role ID from different workspace
		t.Run("by role ID", func(t *testing.T) {
			req := handler.Request{
				KeyId: validKeyID,
				Roles: []string{otherRoleID},
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				headers,
				req,
			)

			require.Equal(t, 404, res.Status)
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
			require.Equal(t, "Not Found", res.Body.Error.Title)
			require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Role '%s' was not found", otherRoleID))
			require.Equal(t, 404, res.Body.Error.Status)
		})

		// Test with role name from different workspace
		t.Run("by role name", func(t *testing.T) {
			req := handler.Request{
				KeyId: validKeyID,
				Roles: []string{otherRoleName},
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				headers,
				req,
			)

			require.Equal(t, 404, res.Status)
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
			require.Equal(t, "Not Found", res.Body.Error.Title)
			require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Role '%s' was not found", otherRoleName))
			require.Equal(t, 404, res.Body.Error.Status)
		})
	})

	// Test case for multiple non-existent roles (first one should fail)
	t.Run("multiple roles with first one non-existent", func(t *testing.T) {
		nonExistentRoleID := "role_first_nonexistent"

		req := handler.Request{
			KeyId: validKeyID,
			Roles: []string{nonExistentRoleID, validRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Role '%s' was not found", nonExistentRoleID))
		require.Equal(t, 404, res.Body.Error.Status)
	})

	// Test case for key with valid format but doesn't exist
	t.Run("valid key format but non-existent", func(t *testing.T) {
		// Use proper format but non-existent key
		validFormattedKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId: validFormattedKeyID,
			Roles: []string{validRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
		require.Equal(t, 404, res.Body.Error.Status)
	})

	// Test case for role with valid format but doesn't exist
	t.Run("valid role format but non-existent", func(t *testing.T) {
		// Use proper format but non-existent role
		validFormattedRoleID := uid.New(uid.TestPrefix)

		req := handler.Request{
			KeyId: validKeyID,
			Roles: []string{validFormattedRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Role '%s' was not found", validFormattedRoleID))
		require.Equal(t, 404, res.Body.Error.Status)
	})
}
