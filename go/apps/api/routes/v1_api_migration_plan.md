# API & Permissions Migration Plan: TypeScript to Go

This document outlines the plan to migrate the `/v1/api.XXX` and `/v1/permissions.XXX` routes from TypeScript to Go. We'll ensure the new Go implementations match the existing TypeScript functionality while following Go best practices and integrating with the existing Go codebase structure.

## Overview

We need to migrate the following endpoints from v1 to v2:

### API Endpoints
- `/v1/apis.createApi` → `/v2/apis.createApi` ✅ (already implemented)
- `/v1/apis.deleteApi` → `/v2/apis.deleteApi` ✅ (implemented)
- `/v1/apis.getApi` → `/v2/apis.getApi` ✅ (implemented)
- `/v1/apis.listKeys` → `/v2/apis.listKeys` ✅ (implemented)

### Permission Endpoints (Priority Order)
1. `/v1/permissions.createPermission` → `/v2/permissions.createPermission`
2. `/v1/permissions.getPermission` → `/v2/permissions.getPermission`
3. `/v1/permissions.listPermissions` → `/v2/permissions.listPermissions`
4. `/v1/permissions.deletePermission` → `/v2/permissions.deletePermission`
5. `/v1/permissions.createRole` → `/v2/permissions.createRole` 
6. `/v1/permissions.getRole` → `/v2/permissions.getRole`
7. `/v1/permissions.listRoles` → `/v2/permissions.listRoles`
8. `/v1/permissions.deleteRole` → `/v2/permissions.deleteRole`

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
- [x] `/v2/apis.listKeys` - Implemented successfully
- [x] `/v2/apis.deleteApi` - Implemented successfully

## API Endpoints Completion

All planned API endpoints have been successfully migrated from TypeScript to Go:

- `/v2/apis.createApi` - Creates a new API with the provided name
- `/v2/apis.getApi` - Retrieves API details by ID
- `/v2/apis.listKeys` - Lists all keys associated with an API with pagination
- `/v2/apis.deleteApi` - Deletes an API and all associated keys (with delete protection)

Each endpoint includes comprehensive test coverage:
- Success cases (200 responses)
- Validation tests (400 responses)
- Authentication tests (401 responses)
- Authorization tests (403 responses)
- Not Found tests (404 responses)
- Special cases (e.g., 429 for delete-protected APIs)

## Permissions Migration Plan

Now we'll focus on migrating the permission-related endpoints. The permission system manages roles and permissions that can be assigned to API keys.

### Understanding the Permission System

Permissions in Unkey are organized as follows:
1. **Permissions** - Individual capabilities that can be assigned to keys (e.g., "read:api", "write:key")
2. **Roles** - Collections of permissions that can be assigned to keys
3. **Workspaces** - Permissions and roles are scoped to workspaces

The database schema includes the following tables:
- `permissions` - Contains individual permissions with name, description, etc.
- `roles` - Contains roles with name, description, etc.
- `role_permissions` - Junction table linking roles to permissions
- `key_permissions` - Links permissions directly to keys
- `key_roles` - Links roles to keys

### Migration Strategy for Permission Routes

For each permission endpoint, we'll follow the same approach as with API endpoints:

1. Study the TypeScript implementation
2. Create Go implementation following established patterns
3. Write comprehensive tests
4. Ensure validation, authorization, and audit logs match existing implementation

### Implementation Plan for Permission Endpoints

#### `/v2/permissions.createPermission`
- Create a new permission in a workspace
- Handle permission naming constraints and uniqueness
- Add audit logging for permission creation
- Implementation steps:
  1. Verify root key authorization with `rbac.*.create_permission` permission
  2. Validate permission name format (follow identifier pattern)
  3. Create permission record in database with unique ID
  4. Handle potential duplicate name errors (return 409 Conflict)
  5. Create audit log entry for permission creation
  6. Return the created permission ID

#### `/v2/permissions.getPermission`
- Retrieve permission details by ID
- Enforce workspace-level access control
- Implementation steps:
  1. Verify root key authorization with `rbac.*.read_permission` permission
  2. Query database for permission by ID
  3. Verify permission belongs to authorized workspace
  4. Return permission details (ID, name, description, etc.)

#### `/v2/permissions.listPermissions`
- List all permissions in a workspace
- Support pagination
- Implementation steps:
  1. Verify root key authorization with `rbac.*.read_permission` permission
  2. Query database for permissions in authorized workspace
  3. Implement pagination with cursor-based approach
  4. Return paginated list of permissions with total count

#### `/v2/permissions.deletePermission`
- Delete a permission
- Handle potential dependencies in roles
- Include audit logging
- Implementation steps:
  1. Verify root key authorization with `rbac.*.delete_permission` permission
  2. Start database transaction
  3. Delete all related records in `role_permissions` junction table
  4. Delete all related records in `key_permissions` table
  5. Delete the permission record
  6. Create audit log entry for permission deletion
  7. Commit transaction
  8. Return success response

#### `/v2/permissions.createRole`
- Create a new role with associated permissions
- Validate permission IDs
- Add audit logging
- Implementation steps:
  1. Verify root key authorization with `rbac.*.create_role` permission
  2. Validate role name format
  3. Start database transaction
  4. Create role record in database with unique ID
  5. Handle potential duplicate name errors (return 409 Conflict)
  6. If permission IDs provided, validate they exist and belong to workspace
  7. Create role-permission relationships in junction table if needed
  8. Create audit log entry for role creation
  9. Commit transaction
  10. Return the created role ID

#### `/v2/permissions.getRole`
- Retrieve role details by ID
- Include associated permissions
- Implementation steps:
  1. Verify root key authorization with `rbac.*.read_role` permission
  2. Query database for role by ID with joined permissions
  3. Verify role belongs to authorized workspace
  4. Return role details (ID, name, description) and associated permissions

#### `/v2/permissions.listRoles`
- List all roles in a workspace
- Support pagination
- Include permission details
- Implementation steps:
  1. Verify root key authorization with `rbac.*.read_role` permission
  2. Query database for roles in authorized workspace with joined permissions
  3. Implement pagination with cursor-based approach
  4. Format response to include role details and associated permissions
  5. Return paginated list of roles with total count

#### `/v2/permissions.deleteRole`
- Delete a role
- Handle relationship cleanup
- Include audit logging
- Implementation steps:
  1. Verify root key authorization with `rbac.*.delete_role` permission
  2. Start database transaction
  3. Delete all related records in `role_permissions` junction table
  4. Delete all related records in `key_roles` table
  5. Delete the role record
  6. Create audit log entry for role deletion
  7. Commit transaction
  8. Return success response

### Progress Tracking

- [x] `/v2/permissions.createPermission` ✅
  - [x] Add OpenAPI definitions
  - [x] Create handler implementation
  - [x] Implement validation and error handling
  - [x] Add audit logging
  - [x] Write tests (200, 400, 401, 403, 409)
  
  #### `/v2/permissions.getPermission` ✅
  - [x] Add OpenAPI definitions
  - [x] Create handler implementation
  - [x] Implement validation and error handling
  - [x] Write tests (200, 400, 401, 403, 404)
  
  #### `/v2/permissions.listPermissions` ✅
  - [x] Add OpenAPI definitions
  - [x] Create handler implementation with pagination
  - [x] Implement validation and error handling
  - [x] Write tests (200, 400, 401, 403)
  
  #### `/v2/permissions.deletePermission` ✅
  - [x] Add OpenAPI definitions
  - [x] Create handler with transaction support
  - [x] Implement validation and error handling
  - [x] Add audit logging
  - [x] Write tests (200, 400, 401, 403, 404)
  
#### `/v2/permissions.createRole` ✅
- [x] Add OpenAPI definitions
- [x] Create handler with transaction support
- [x] Implement permission validation
- [x] Add audit logging
- [x] Write tests (200, 400, 401, 403, 409)
  
#### `/v2/permissions.getRole` ✅
- [x] Add OpenAPI definitions
- [x] Create handler with permission joins
- [x] Implement validation and error handling
- [x] Write tests (200, 400, 401, 403, 404)
  
#### `/v2/permissions.listRoles` ✅
- [x] Add OpenAPI definitions
- [x] Create handler with permission joins and pagination
- [x] Implement validation and error handling
- [x] Write tests (200, 400, 401, 403)
  
#### `/v2/permissions.deleteRole` ✅
- [x] Add OpenAPI definitions
- [x] Create handler with transaction support
- [x] Implement validation and error handling
- [x] Implement audit logging
- [x] Write tests (200, 400, 401, 403, 404)

## Learnings from Initial Implementation

1. Follow the RPC style API pattern with POST methods for all operations
2. Always add Content-Type headers in tests
3. Handle error responses and status codes consistently
4. Test all error cases, especially authentication and authorization
5. Use proper database queries and error handling
6. Be precise with RBAC permissions checks
7. Always update the OpenAPI schema before implementation
8. For endpoints with pagination, return appropriate cursor values and total counts
9. Handle nullability carefully when mapping database fields to response objects
10. When dealing with optional parameters, provide sensible defaults (e.g., limit=100)
11. Use transactions for operations that modify multiple related tables
12. Clear caches after modifying data to ensure consistency
13. Implement audit logging for all modification operations
14. Handle special cases like delete protection with appropriate status codes
15. Understand and implement relationships between entities (e.g., roles and permissions)
16. For permission-related operations, ensure proper checks for workspace boundaries
17. When deleting entities with relationships, handle cascading deletes within transactions
18. Handle unique constraint violations with appropriate error responses (409 Conflict)
19. For role and permission operations, be careful with case sensitivity in name matching
20. Always add complete error test coverage for edge cases like non-existent permissions
