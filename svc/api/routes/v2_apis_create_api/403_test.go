package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
)

// TestCreateApi_Forbidden verifies that API creation requests are properly
// rejected when the authenticated user lacks the required permissions. This test
// ensures that RBAC (Role-Based Access Control) is correctly enforced and that
// users without api.*.create_api permission receive 403 Forbidden responses.
func TestCreateApi_Forbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a root key with insufficient permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity") // Not api.*.create_api
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This test validates that a root key with valid authentication but
	// insufficient permissions (lacking api.*.create_api) is properly rejected
	// with a 403 status code, ensuring permission boundaries are enforced.
	t.Run("insufficient permissions", func(t *testing.T) {
		req := handler.Request{
			Name: "test-api",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
	})

	// This test validates various permission combinations to ensure that only
	// root keys with the exact api.*.create_api permission can create APIs, while
	// keys with other permissions or insufficient permissions are rejected.
	t.Run("permission combinations", func(t *testing.T) {
		testCases := []struct {
			name        string
			permissions []string
			shouldPass  bool
		}{
			{name: "specific permission", permissions: []string{"api.*.create_api"}, shouldPass: true},
			{name: "specific permission and more", permissions: []string{"some.other.permission", "xxx", "api.*.create_api", "another.permission"}, shouldPass: true},
			{name: "insufficient permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
			{name: "unrelated permission", permissions: []string{"identity.*.create_identity"}, shouldPass: false},
		}

		// Each test case validates a specific permission scenario to ensure
		// proper RBAC enforcement across different permission combinations.
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a root key with the specific permissions
				permRootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, tc.permissions...)
				permHeaders := http.Header{
					"Content-Type":  {"application/json"},
					"Authorization": {fmt.Sprintf("Bearer %s", permRootKey)},
				}

				req := handler.Request{
					Name: "test-api-permissions",
				}

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, permHeaders, req)

				if tc.shouldPass {
					require.Equal(t, 200, res.Status, "Expected 200 for permission: %v, got: %s", tc.permissions, res.RawBody)
					require.NotEmpty(t, res.Body.Data.ApiId)

					// Verify the API in the database
					api, err := db.Query.FindApiByID(context.Background(), h.DB.RO(), res.Body.Data.ApiId)
					require.NoError(t, err)
					require.Equal(t, req.Name, api.Name)
				} else {
					require.Equal(t, http.StatusForbidden, res.Status, "Expected 403 for permission: %v, got: %s", tc.permissions, res.RawBody)
				}
			})
		}
	})
}
