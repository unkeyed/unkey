# API Migration Plan: TypeScript to Go

This document outlines the plan to migrate the `/v1/api.XXX` routes from TypeScript to Go. We'll ensure the new Go implementations match the existing TypeScript functionality while following Go best practices and integrating with the existing Go codebase structure.

## Overview

We need to migrate the following API endpoints from v1 to v2:

- `/v1/apis.createApi` → `/v2/apis.createApi` ✅ (already implemented)
- `/v1/apis.deleteApi` → `/v2/apis.deleteApi`
- `/v1/apis.getApi` → `/v2/apis.getApi` ✅ (implemented)
- `/v1/apis.listKeys` → `/v2/apis.listKeys`

## Migration Strategy

For each endpoint, we'll:

1. Study the TypeScript implementation
2. Create Go implementation following established patterns
3. Write comprehensive tests
4. Ensure validation, authorization, and audit logs match existing implementation

## Implementation Guide

### Understanding the Architecture

The Go implementation follows these key patterns:

1. **RPC-style API**: All endpoints are implemented as POST operations, regardless of whether they're retrieving or modifying data
2. **Directory Structure**: Each endpoint gets its own directory under `go/apps/api/routes/`
3. **OpenAPI Types**: Request and response types are defined in `go/apps/api/openapi/openapi.json` and generated into Go structs

### Implementation Steps for Each Endpoint

1. **Study TypeScript Implementation**
   - Read the TypeScript code to understand the functionality
   - Note any business logic, validation, or security checks
   - Identify request/response schemas

2. **Add OpenAPI Definition**
   - Add the endpoint definition to `go/apps/api/openapi/openapi.json`
   - Include request/response schemas matching the TypeScript version
   - Run the generator (`go generate ./go/apps/api/openapi/`) to create Go structs

3. **Create Directory Structure**
   - Create a new directory for the endpoint (e.g., `go/apps/api/routes/v2_apis_get_api/`)
   - Create handler.go file for implementation
   - Create test files (200_test.go, 400_test.go, 401_test.go, 403_test.go, 404_test.go)

4. **Implement Handler**
   - Create a Services struct with required dependencies
   - Implement the handler function with:
     - Authentication using `svc.Keys.VerifyRootKey`
     - Request validation
     - RBAC permission checks
     - Database operations
     - Return properly formatted response

5. **Write Tests**
   - Success tests (200 responses) for valid requests
   - Validation tests (400 responses) for invalid inputs
   - Authentication tests (401 responses) for invalid credentials
   - Authorization tests (403 responses) for insufficient permissions
   - Not Found tests (404 responses) for non-existent resources

6. **Update register.go**
   - Add the new route to `go/apps/api/routes/register.go`
   - Include required service dependencies

### Code Structure

#### Handler Template

```go
package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2ApisXxxRequestBody
type Response = openapi.V2ApisXxxResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	// Add other services as needed
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/apis.xxx", func(ctx context.Context, s *zen.Session) error {
		// 1. Authentication
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// 2. Request validation
		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		// 3. Permission check
		permissions, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				// Add appropriate permissions
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissions.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissions.Message, permissions.Message),
			)
		}

		// 4. Business logic
		// Implement core functionality here

		// 5. Return response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.XxxResponseData{
				// Fill response data
			},
		})
	})
}
```

#### Test Template

```go
package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_xxx"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestSuccessCase(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		// Add other services as needed
	})

	h.Register(route)

	// Create test data

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "required.permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case
	t.Run("successful operation", func(t *testing.T) {
		req := handler.Request{
			// Fill request data
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		// Add more assertions as needed
	})
}
```

### Common Gotchas and Solutions

1. **Content-Type Header**: Always include `"Content-Type": {"application/json"}` in test headers for POST requests

2. **Error Codes**: Use the correct error codes package:
   - `codes.Auth.Authorization.InsufficientPermissions.URN()` for permission errors
   - `codes.Data.Api.NotFound.URN()` for not found errors
   - `codes.App.Validation.InvalidInput.URN()` for validation errors

3. **Test Response Types**: Use the right type for test responses:
   - `openapi.BadRequestErrorResponse` for 400 errors
   - `openapi.UnauthorizedErrorResponse` for 401 errors
   - `openapi.ForbiddenErrorResponse` for 403 errors
   - `openapi.NotFoundErrorResponse` for 404 errors

4. **Error Messages**: Don't test for exact error messages; they might change. Instead, test for:
   - HTTP status code
   - Error type URL
   - Partial content of error detail

5. **SQL Null Values**: Use `sql.NullString{Valid: true, String: value}` or similar for nullable DB fields

### Testing Requirements

Each endpoint should have tests for:

1. **Success scenarios (200_test.go)**
   - Basic success case
   - Different permission combinations
   - Edge cases with valid inputs

2. **Validation errors (400_test.go)**
   - Missing required fields
   - Invalid field formats or values
   - Schema validation failures

3. **Authentication errors (401_test.go)**
   - Missing authentication
   - Invalid token
   - Malformed authorization header

4. **Authorization errors (403_test.go)**
   - Insufficient permissions
   - Permissions for different resources
   - Wrong workspace

5. **Not found errors (404_test.go)**
   - Non-existent resources
   - Resources from other workspaces
   - Deleted resources (where applicable)

## Progress Tracking

- [x] `/v2/apis.createApi` - Already implemented
- [x] `/v2/apis.getApi` - Implemented successfully
- [ ] `/v2/apis.listKeys` - Next to implement
- [ ] `/v2/apis.deleteApi` - Last to implement

## Remaining Work

### `/v2/apis.listKeys` Endpoint

Similar to the `getApi` endpoint but with pagination and filtering. Key steps:

1. Study TypeScript implementation in `/apps/api/src/routes/v1_apis_listKeys.ts`
2. Add OpenAPI definition in `openapi.json` with request/response schema
3. Create directory and files for implementation and tests
4. Implement handler with pagination support
5. Write comprehensive tests
6. Register route in `register.go`

### `/v2/apis.deleteApi` Endpoint

More complex due to potential cascading effects. Key steps:

1. Study TypeScript implementation in `/apps/api/src/routes/v1_apis_deleteApi.ts`
2. Add OpenAPI definition in `openapi.json`
3. Create directory and files for implementation and tests
4. Implement handler with proper transaction support and audit logging
5. Handle delete protection if present
6. Write comprehensive tests
7. Register route in `register.go`

## Learnings from Initial Implementation

1. Follow the RPC style API pattern with POST methods for all operations
2. Always add Content-Type headers in tests
3. Handle error responses and status codes consistently
4. Test all error cases, especially authentication and authorization
5. Use proper database queries and error handling
6. Be precise with RBAC permissions checks
7. Always update the OpenAPI schema before implementation
