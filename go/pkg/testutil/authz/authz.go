package authz

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Test401 runs a comprehensive suite of authentication failure tests.
// It tests all common scenarios where requests should return 401 Unauthorized.
//
// Type parameters:
//   - TReq: The request type for the endpoint
//   - TRes: The response type for the endpoint
//
// Parameters:
//   - t: The testing context
//   - setupHandler: Function to create the handler with test dependencies
//   - createRequest: Function to create a valid request body
//
// Example usage:
//
//	func TestCreateApiUnauthorized(t *testing.T) {
//	    authz.Test401[handler.Request, handler.Response](t,
//	        func(h *testutil.Harness) zen.Route {
//	            return &handler.Handler{DB: h.DB, Keys: h.Keys, Logger: h.Logger}
//	        },
//	        func() handler.Request {
//	            return handler.Request{Name: "test-api"}
//	        },
//	    )
//	}
func Test401[TReq any, TRes any](
	t *testing.T,
	setupHandler func(*testutil.Harness) zen.Route,
	createRequest func() TReq,
) {
	t.Helper()

	testCases := []struct {
		name           string
		authorization  string
		expectedStatus int
	}{
		{
			name:           "invalid bearer token",
			authorization:  "Bearer invalid_token_12345",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "nonexistent key",
			authorization:  fmt.Sprintf("Bearer %s", uid.New(uid.KeyPrefix)),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "bearer with extra spaces",
			authorization:  "Bearer   invalid_key_with_spaces   ",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing authorization header",
			authorization:  "",
			expectedStatus: http.StatusBadRequest, // Missing auth typically returns 400
		},
		{
			name:           "empty authorization header",
			authorization:  " ",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed authorization header - no Bearer prefix",
			authorization:  "invalid_token_without_bearer",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed authorization header - Bearer only",
			authorization:  "Bearer",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := setupHandler(h)
			h.Register(route)

			headers := http.Header{
				"Content-Type": {"application/json"},
			}
			if tc.authorization != "" {
				headers.Set("Authorization", tc.authorization)
			}

			req := createRequest()
			res := testutil.CallRoute[TReq, openapi.UnauthorizedErrorResponse](h, route, headers, req)

			require.Equal(t, tc.expectedStatus, res.Status,
				"expected %d, got %d, body: %s", tc.expectedStatus, res.Status, res.RawBody)
			require.NotNil(t, res.Body)
		})
	}
}

// Test403 runs a comprehensive suite of authorization failure tests.
// It tests all common scenarios where requests should return 403 Forbidden.
//
// Type parameters:
//   - TReq: The request type for the endpoint
//   - TRes: The response type for the endpoint
//
// Parameters:
//   - t: The testing context
//   - config: Configuration for the permission tests
//
// The test suite automatically generates test cases for:
//   - No permissions
//   - Wrong permissions (different action)
//   - Unrelated permissions
//   - Resource-specific permissions (if applicable)
//   - Cross-workspace scenarios
//   - Permission combinations
//
// Example usage:
//
//	func TestCreateApiForbidden(t *testing.T) {
//	    authz.Test403(t,
//	        authz.PermissionTestConfig[handler.Request, handler.Response]{
//	            SetupHandler: func(h *testutil.Harness) zen.Route {
//	                return &handler.Handler{DB: h.DB, Keys: h.Keys, Logger: h.Logger}
//	            },
//	            RequiredPermissions: []string{"api.*.create_api"},
//	            CreateRequest: func(res authz.TestResources) handler.Request {
//	                return handler.Request{Name: "test-api"}
//	            },
//	        },
//	    )
//	}
func Test403[TReq any, TRes any](
	t *testing.T,
	config PermissionTestConfig[TReq, TRes],
) {
	t.Helper()

	h := testutil.NewHarness(t)
	route := config.SetupHandler(h)
	h.Register(route)

	// Setup resources
	var resources TestResources
	if config.SetupResources != nil {
		resources = config.SetupResources(h)
	} else {
		resources = TestResources{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		}
	}

	// Generate standard test cases
	testCases := generateStandardPermissionTests[TReq](config.RequiredPermissions, resources)

	// Add any additional custom test cases
	for _, customTest := range config.AdditionalPermissionTests {
		testCases = append(testCases, customTest)
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Create root key with specified permissions
			rootKey := h.CreateRootKey(resources.WorkspaceID, tc.Permissions...)

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			req := config.CreateRequest(resources)
			if tc.ModifyRequest != nil {
				req = tc.ModifyRequest(req, resources)
			}

			res := testutil.CallRoute[TReq, openapi.ForbiddenErrorResponse](h, route, headers, req)

			require.Equal(t, tc.ExpectedStatus, res.Status,
				"expected %d, got %d, body: %s", tc.ExpectedStatus, res.Status, res.RawBody)

			if tc.ExpectedStatus == http.StatusForbidden {
				require.NotNil(t, res.Body)
				require.NotNil(t, res.Body.Error)
				if tc.ValidateError != nil {
					tc.ValidateError(t, res.Body.Error.Detail)
				}
			}
		})
	}

	// Test permission combinations (some should pass, some should fail)
	t.Run("permission combinations", func(t *testing.T) {
		testPermissionCombinations(t, h, route, config, resources)
	})
}

// generateStandardPermissionTests creates standard test cases based on required permissions
func generateStandardPermissionTests[TReq any](requiredPermissions []string, resources TestResources) []PermissionTestCase[TReq] {
	testCases := []PermissionTestCase[TReq]{
		{
			Name:           "no permissions",
			Permissions:    []string{},
			ExpectedStatus: http.StatusForbidden,
		},
		{
			Name:           "unrelated permission",
			Permissions:    []string{"completely.unrelated.permission"},
			ExpectedStatus: http.StatusForbidden,
		},
	}

	// For each required permission, generate related test cases
	for _, perm := range requiredPermissions {
		parts := strings.Split(perm, ".")
		if len(parts) >= 3 {
			// Generate wrong action test (e.g., api.*.read instead of api.*.create)
			wrongAction := parts[0] + "." + parts[1] + ".wrong_action"
			testCases = append(testCases, PermissionTestCase[TReq]{
				Name:           fmt.Sprintf("wrong action for %s", parts[0]),
				Permissions:    []string{wrongAction},
				ExpectedStatus: http.StatusForbidden,
			})

			// If permission uses wildcard and we have a specific resource ID, test mismatch
			if parts[1] == "*" && resources.ApiID != "" {
				otherID := uid.New(uid.APIPrefix)
				specificWrong := parts[0] + "." + otherID + "." + parts[2]
				testCases = append(testCases, PermissionTestCase[TReq]{
					Name:           "permission for different resource",
					Permissions:    []string{specificWrong},
					ExpectedStatus: http.StatusForbidden,
				})
			}
		}
	}

	return testCases
}

// testPermissionCombinations tests various combinations of permissions
func testPermissionCombinations[TReq any, TRes any](
	t *testing.T,
	h *testutil.Harness,
	route zen.Route,
	config PermissionTestConfig[TReq, TRes],
	resources TestResources,
) {
	t.Helper()

	if len(config.RequiredPermissions) == 0 {
		return
	}

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{
			name:        "exact required permission",
			permissions: config.RequiredPermissions,
			shouldPass:  true,
		},
		{
			name:        "required permission plus additional",
			permissions: append([]string{"some.other.permission"}, config.RequiredPermissions...),
			shouldPass:  true,
		},
		{
			name: "only additional permissions",
			permissions: []string{
				"some.other.permission",
				"another.permission",
			},
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(resources.WorkspaceID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			req := config.CreateRequest(resources)
			res := testutil.CallRoute[TReq, TRes](h, route, headers, req)

			if tc.shouldPass {
				require.NotEqual(t, http.StatusForbidden, res.Status,
					"expected success, got %d, body: %s", res.Status, res.RawBody)
				require.NotEqual(t, http.StatusUnauthorized, res.Status,
					"expected success, got %d, body: %s", res.Status, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status,
					"expected 403, got %d, body: %s", res.Status, res.RawBody)
			}
		})
	}
}
