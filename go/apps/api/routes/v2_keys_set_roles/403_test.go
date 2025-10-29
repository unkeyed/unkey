package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_roles"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestAuthorizationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		LiveKeyCache: h.Caches.LiveKeyByID,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create test data using testutil helper
	defaultPrefix := "test"
	defaultBytes := int32(16)
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		DefaultPrefix: &defaultPrefix,
		DefaultBytes:  &defaultBytes,
	})

	keyName := "Test Key"
	roleDescription := "Test role"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Name:        &keyName,
		Roles: []seed.CreateRoleRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "test-role",
				Description: &roleDescription,
			},
		},
	})
	keyID := keyResponse.KeyID
	roleID := keyResponse.RolesIds[0]

	// Test case for insufficient permissions - missing update_key
	t.Run("missing update_key permission", func(t *testing.T) {
		// Create a root key with some permissions but not update_key
		rootKey := h.CreateRootKey(workspace.ID, "api.*.create_key") // Only has create, not update

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
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
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for no permissions at all
	t.Run("no permissions", func(t *testing.T) {
		// Create a root key with no permissions
		rootKey := h.CreateRootKey(workspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
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
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

}
