package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
)

// TestCreateApi_Forbidden verifies that API creation requests are rejected when
// the authenticated user lacks the required permissions.
func TestCreateApi_Forbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a root key with insufficient permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/identities/*#write", h.Resources().UserWorkspace.ID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This test validates that a root key with valid authentication but
	// insufficient permissions are properly rejected
	// with a 403 status code, ensuring permission boundaries are enforced.
	t.Run("insufficient permissions", func(t *testing.T) {
		req := handler.Request{
			Name: "test-api",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
	})

	// This test validates permission combinations for API creation.
	t.Run("permission combinations", func(t *testing.T) {
		testCases := []struct {
			name        string
			permissions []string
			shouldPass  bool
		}{}

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

// TestCreateApi_ForbiddenForOtherProjectGrant verifies that a keyspace grant on
// another project does not cover the default project, and that the rejected
// request does not create the default project.
func TestCreateApi_ForbiddenForOtherProjectGrant(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	grant := rbac.U(
		urn.New().Workspace(workspaceID).Project(uid.New(uid.ProjectPrefix)).Keyspace("*"),
		permissions.Write,
	).Value
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", h.CreateRootKey(workspaceID, grant))},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Name: "KEBAP"})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)

	_, found, err := projects.FindDefaultProject(ctx, h.DB.RO(), workspaceID)
	require.NoError(t, err)
	require.False(t, found, "a rejected request must not create the default project")
}
