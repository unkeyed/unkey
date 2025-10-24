# Authorization Test Helpers

This package provides generic, reusable test helpers for authentication (401) and authorization (403) testing across all API endpoints.

## Overview

Instead of writing repetitive test code for each endpoint, these helpers automatically test all common authentication and authorization scenarios with minimal configuration.

## Benefits

- **Consistent test coverage**: All endpoints test the same scenarios
- **Single source of truth**: Update test logic in one place
- **Type-safe with generics**: Compile-time type checking for requests and responses
- **Extensible**: Easy to add custom test cases

## Usage

### Basic 401 Test (Authentication Failures)

For endpoints that don't need existing resources:

```go
func TestCreateApiUnauthorized(t *testing.T) {
    authz.Test401[handler.Request, handler.Response](t,
        func(h *testutil.Harness) zen.Route {
            return &handler.Handler{
                DB:        h.DB,
                Keys:      h.Keys,
                Logger:    h.Logger,
                Auditlogs: h.Auditlogs,
            }
        },
        func() handler.Request {
            return handler.Request{
                Name: "test-api",
            }
        },
    )
}
```

This automatically tests:

- Invalid bearer token
- Nonexistent key
- Bearer with extra spaces
- Missing authorization header
- Empty authorization header
- Malformed header - no Bearer prefix
- Malformed header - Bearer only

### Basic 403 Test (Authorization Failures)

For simple endpoints without resource dependencies:

```go
func TestCreateApiForbidden(t *testing.T) {
    authz.Test403(t,
        authz.PermissionTestConfig[handler.Request, handler.Response]{
            SetupHandler: func(h *testutil.Harness) zen.Route {
                return &handler.Handler{
                    DB:        h.DB,
                    Keys:      h.Keys,
                    Logger:    h.Logger,
                    Auditlogs: h.Auditlogs,
                }
            },
            RequiredPermissions: []string{"api.*.create_api"},
            CreateRequest: func(res authz.TestResources) handler.Request {
                return handler.Request{
                    Name: "test-api",
                }
            },
        },
    )
}
```

This automatically tests:

- No permissions
- Unrelated permissions
- Wrong action (e.g., `read` instead of `create`)
- Permission for different resource
- Permission combinations (exact match, with extras, insufficient)

### Advanced 403 Test (With Resource Setup)

For endpoints that need existing resources:

```go
func TestUpdateKeyForbidden(t *testing.T) {
    authz.Test403(t,
        authz.PermissionTestConfig[handler.Request, handler.Response]{
            SetupHandler: func(h *testutil.Harness) zen.Route {
                return &handler.Handler{
                    DB:        h.DB,
                    Keys:      h.Keys,
                    Logger:    h.Logger,
                    Auditlogs: h.Auditlogs,
                    Vault:     h.Vault,
                }
            },
            RequiredPermissions: []string{"api.*.update_key"},

            // Setup creates resources needed for the test
            SetupResources: func(h *testutil.Harness) authz.TestResources {
                api := h.CreateApi(seed.CreateApiRequest{
                    WorkspaceID: h.Resources().UserWorkspace.ID,
                })
                key := h.CreateKey(seed.CreateKeyRequest{
                    WorkspaceID: h.Resources().UserWorkspace.ID,
                    KeyAuthID:   api.KeyAuthID.String,
                })
                otherApi := h.CreateApi(seed.CreateApiRequest{
                    WorkspaceID: h.Resources().UserWorkspace.ID,
                })

                return authz.TestResources{
                    WorkspaceID: h.Resources().UserWorkspace.ID,
                    ApiID:       api.ID,
                    KeyID:       key.KeyID,
                    OtherApiID:  otherApi.ID,
                }
            },

            // Request uses the created resources
            CreateRequest: func(res authz.TestResources) handler.Request {
                return handler.Request{
                    KeyId: res.KeyID,
                    Name:  ptr.String("updated-name"),
                }
            },

            // Optional: Add custom test cases
            AdditionalPermissionTests: []authz.PermissionTestCase[handler.Request]{
                {
                    Name:        "special scenario",
                    Permissions: []string{"some.permission"},
                    ModifyRequest: func(req handler.Request) handler.Request {
                        req.SomeField = "modified"
                        return req
                    },
                    ExpectedStatus: 403,
                },
            },
        },
    )
}
```

## TestResources Structure

The `TestResources` struct provides commonly used resource IDs:

```go
type TestResources struct {
    WorkspaceID      string // Primary workspace ID
    OtherWorkspaceID string // For cross-workspace tests
    ApiID            string // Primary API ID
    OtherApiID       string // For cross-resource tests
    KeyAuthID        string // Primary key auth ID
    KeyID            string // Primary key ID
    IdentityID       string // Primary identity ID
    Custom           map[string]string // For endpoint-specific IDs
}
```

## Configuration Options

### PermissionTestConfig

- **SetupHandler**: Creates the handler with dependencies (required)
- **RequiredPermissions**: List of permissions needed (required)
- **CreateRequest**: Creates a valid request body (required)
- **SetupResources**: Optional, creates test data before running tests
- **AdditionalPermissionTests**: Optional, custom test cases beyond standard ones

### PermissionTestCase

- **Name**: Test case name
- **Permissions**: Permissions to grant
- **ModifyRequest**: Optional, modify the request for this test
- **ExpectedStatus**: Expected HTTP status code
- **ValidateError**: Optional, custom error validation

## Migration Guide

### Before (Old Pattern)

```go
func TestCreateApiUnauthorized(t *testing.T) {
    h := testutil.NewHarness(t)
    route := &handler.Handler{...}
    h.Register(route)

    t.Run("invalid bearer token", func(t *testing.T) {
        headers := http.Header{
            "Content-Type":  {"application/json"},
            "Authorization": {"Bearer invalid_token"},
        }
        req := handler.Request{Name: "test-api"}
        res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
        require.Equal(t, http.StatusUnauthorized, res.Status)
    })

    // ... 6 more similar test cases (50+ lines)
}
```

### After (New Pattern)

```go
func TestCreateApiUnauthorized(t *testing.T) {
    authz.Test401[handler.Request, handler.Response](t,
        func(h *testutil.Harness) zen.Route {
            return &handler.Handler{DB: h.DB, Keys: h.Keys, Logger: h.Logger}
        },
        func() handler.Request {
            return handler.Request{Name: "test-api"}
        },
    )
}
```

## Examples

See these endpoints for reference implementations:

- **Simple**: `apps/api/routes/v2_apis_create_api/*_test.go`
- **Medium**: `apps/api/routes/v2_keys_create_key/*_test.go`
- **Complex**: `apps/api/routes/v2_keys_update_key/*_test.go` (when implemented)

## Remaining Work

To complete the migration:

1. Refactor remaining ~65 test files following the patterns above
2. Each endpoint should take ~5-10 minutes to refactor
3. Run tests after each refactor to ensure no regressions

Endpoints to refactor:

- `v2_keys_*` (30 endpoints)
- `v2_apis_*` (3 remaining)
- `v2_identities_*` (6 endpoints)
- `v2_permissions_*` (8 endpoints)
- `v2_ratelimit_*` (5 endpoints)
