package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestUpdateKeyCorrectPermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		roles []string
	}{
		{
			name:  "wildcard api",
			roles: []string{"api.*.update_key"},
		},
		{
			name:  "specific api",
			roles: []string{}, // Will be filled in with specific API ID
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := testutil.NewHarness(t)

			route := &handler.Handler{
				DB:           h.DB,
				Keys:         h.Keys,
				Logger:       h.Logger,
				Auditlogs:    h.Auditlogs,
				KeyCache:     h.Caches.VerificationKeyByHash,
				UsageLimiter: h.UsageLimiter,
			}

			h.Register(route)

			// Create API using helper
			api := h.CreateApi(seed.CreateApiRequest{
				WorkspaceID: h.Resources().UserWorkspace.ID,
			})

			// Create key using helper
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				Name:        ptr.P("test"),
			})

			// Set up permissions
			roles := tc.roles
			if tc.name == "specific api" {
				roles = []string{fmt.Sprintf("api.%s.update_key", api.ID)}
			}

			rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, roles...)

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			req := handler.Request{
				KeyId:   keyResponse.KeyID,
				Enabled: ptr.P(false),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
			require.NotNil(t, res.Body)
		})
	}
}

func TestUpdateKeyInsufficientPermissions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create API using helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create key using helper
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Name:        ptr.P("test"),
	})

	// Create root key with insufficient permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key") // Wrong permission

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		KeyId:   keyResponse.KeyID,
		Enabled: ptr.P(false),
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, 403, res.Status)
	require.NotNil(t, res.Body)
}

func TestUpdateKeyCrossWorkspaceIsolation(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create API using helper in user workspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create key using helper in user workspace
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Name:        ptr.P("test"),
	})

	// Create different workspace
	otherWorkspace := h.CreateWorkspace()

	// Create root key for other workspace
	rootKey := h.CreateRootKey(otherWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		KeyId:   keyResponse.KeyID,
		Enabled: ptr.P(false),
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, 404, res.Status)
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Detail, "The specified key was not found")
}
