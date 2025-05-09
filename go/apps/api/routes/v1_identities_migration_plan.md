# Identity Routes Migration Plan: TypeScript to Go

This document outlines the plan to migrate the `/v1/identities.XXX` routes from TypeScript to Go. We'll ensure the new Go implementations match the existing TypeScript functionality while following Go best practices and integrating with the existing Go codebase structure.

## Overview

We need to migrate the following endpoints from v1 to v2:

### Identity Endpoints
- `/v1/identities.createIdentity` → `/v2/identities.createIdentity`
- `/v1/identities.getIdentity` → `/v2/identities.getIdentity`
- `/v1/identities.listIdentities` → `/v2/identities.listIdentities`
- `/v1/identities.updateIdentity` → `/v2/identities.updateIdentity`
- `/v1/identities.deleteIdentity` → `/v2/identities.deleteIdentity`

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

2. **Implement Go Handler**
   - Create a new directory for the endpoint
   - Create a route handler with the same validation and logic
   - Ensure proper error handling
   - Follow established patterns from existing Go handlers

3. **Add Route Registration**
   - Register the new route in `go/apps/api/app.go`

4. **Write Tests**
   - Happy path tests (200 responses)
   - Validation tests (400 responses)
   - Authentication tests (401 responses)
   - Authorization tests (403 responses)
   - Not Found tests (404 responses)
   - Special cases (e.g., 409 for duplicate identities)

### Code Structure

#### Handler Template

Each endpoint's package will contain the following files:
- `handler.go` - Main handler implementation
- `handler_test.go` - Unit tests for the handler

The handler will use a structure similar to:

```go
type Services struct {
    Logger      log.Logger
    DB          *database.DB
    Keys        *service.KeyService
    Identities  *service.IdentityService
    Ratelimits  *service.RatelimitService
}

func New(services Services) func(w http.ResponseWriter, r *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse request
        // 2. Validate input
        // 3. Authenticate user
        // 4. Authorize action
        // 5. Perform business logic
        // 6. Record audit logs
        // 7. Return response
    }
}
```

#### Test Template

Tests should cover:
- Success cases with valid input
- Invalid input (400)
- Unauthorized access (401)
- Insufficient permissions (403)
- Resource not found (404)
- Conflict cases (409)

### Common Gotchas and Solutions

1. **Authentication/Authorization**: Use the `rootKeyAuth` function for consistent auth patterns
2. **Error Handling**: Use `UnkeyApiError` with appropriate status codes
3. **Database Errors**: Handle common DB errors (e.g., duplicates, constraints)
4. **Audit Logging**: Ensure each endpoint logs appropriate audit events
5. **Transactions**: Use transactions for operations that modify multiple tables

### Testing Requirements

- Each endpoint should have comprehensive test coverage
- Tests should verify both success and error scenarios
- Integration tests should validate the end-to-end functionality
- Use mocks where appropriate to isolate components

## Implementation Plan for Identity Endpoints

### `/v2/identities.createIdentity`

Creates a new identity with optional metadata and rate limits.

**Key functionality:**
- Create identity with unique externalId
- Store optional metadata (with size validation)
- Create optional rate limits
- Record audit logs

### `/v2/identities.getIdentity`

Retrieves identity details by ID or externalId.

**Key functionality:**
- Fetch identity by ID or externalId
- Include metadata and rate limits in response
- Handle not found cases

### `/v2/identities.listIdentities`

Lists identities for a workspace with pagination.

**Key functionality:**
- List identities with pagination
- Filter by environment
- Sort options

### `/v2/identities.updateIdentity`

Updates an existing identity's metadata and/or rate limits.

**Key functionality:**
- Update identity metadata
- Add/update/remove rate limits
- Record audit logs for all changes

### `/v2/identities.deleteIdentity`

Deletes an identity and associated rate limits.

**Key functionality:**
- Delete identity record
- Clean up associated rate limits
- Record audit logs

## Progress Tracking

- [ ] `/v2/identities.createIdentity`
- [ ] `/v2/identities.getIdentity`
- [ ] `/v2/identities.listIdentities`
- [ ] `/v2/identities.updateIdentity`
- [ ] `/v2/identities.deleteIdentity`

## Learnings and Best Practices

As we implement these endpoints, we'll document any learnings, gotchas, or best practices here to help with future migrations.