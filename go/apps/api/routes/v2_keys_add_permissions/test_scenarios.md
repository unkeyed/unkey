# Test Scenarios for v2_keys_add_permissions

This document outlines the comprehensive test scenarios for the `/v2/keys.addPermissions` endpoint.

## Overview

The endpoint allows adding permissions directly to an API key. It's an idempotent operation that only adds permissions that aren't already assigned to the key.

## Test Coverage

### 200 - Success Scenarios

#### Basic Operations
- **Add single permission by ID**: Verifies that a single permission can be added using its permission ID
- **Add single permission by name**: Verifies that a single permission can be added using its permission name
- **Add multiple permissions**: Verifies that multiple permissions can be added in a single request using mixed ID and name references

#### Edge Cases
- **Idempotent operation**: Verifies that adding the same permission twice results in the same state (no duplicates)
- **Add to key with existing permissions**: Verifies that new permissions are added to keys that already have some permissions
- **Response sorting**: Verifies that the response permissions are sorted alphabetically by name for consistency

#### Verification Points
- Response status is 200
- Response includes `requestId` in metadata
- Response data contains all direct permissions assigned to the key
- Database state matches the response
- Audit logs are created for permission additions
- Only new permissions trigger database operations (idempotent behavior)

### 400 - Validation Errors

#### Request Validation
- **Empty keyId**: Request with empty or missing key ID
- **Empty permissions array**: Request with no permissions specified
- **Permission missing both id and name**: Permission object with neither `id` nor `name` specified

#### Data Validation
- **Permission not found by id**: Request references a permission ID that doesn't exist
- **Permission not found by name**: Request references a permission name that doesn't exist
- **Key not found**: Request references a key ID that doesn't exist

### 401 - Authentication Errors

#### Missing/Invalid Authentication
- **Missing authorization header**: Request without Authorization header
- **Invalid bearer token**: Request with malformed or non-existent token
- **Malformed authorization header**: Request with incorrectly formatted Authorization header
- **Empty bearer token**: Request with "Bearer " but no token

#### Key State Issues
- **Disabled root key**: Request using a root key that has been disabled

### 403 - Authorization Errors

#### Insufficient Permissions
- **Root key without required permissions**: Root key missing `api.*.update_key` permission
- **Root key with partial permissions**: Root key with related but insufficient permissions (e.g., `api.read.update_key`)
- **Root key with no permissions**: Root key that has no permissions assigned

#### Workspace Isolation
- **Key belongs to different workspace**: Attempting to modify a key from a different workspace
- **Permission belongs to different workspace**: Attempting to assign a permission from a different workspace

### 404 - Not Found Errors

#### Resource Not Found
- **Key not found**: Specified key ID doesn't exist in the authorized workspace
- **Permission not found by ID**: Specified permission ID doesn't exist in the authorized workspace
- **Permission not found by name**: Specified permission name doesn't exist in the authorized workspace

#### Cross-Workspace Access
- **Permission from different workspace by ID**: Permission exists but belongs to a different workspace (returns 404, not 403, for security)
- **Key from different workspace**: Key exists but belongs to a different workspace (returns 404, not 403, for security)

## Security Considerations

### Workspace Isolation
- Keys and permissions are strictly isolated by workspace
- Cross-workspace access attempts return 404 (not 403) to avoid information disclosure
- Root keys can only operate on resources within their authorized workspace

### Permission Requirements
- Requires `api.*.update_key` permission on the root key
- Wildcard permissions (`*`) are supported for broader access
- Permission hierarchy is enforced (specific permissions don't grant broader access)

### Audit Trail
- All permission additions are logged to the audit system
- Includes actor information (root key), target resources, and operation details
- Failed operations are not logged (only successful additions)

## Performance Considerations

### Database Operations
- Uses transactions to ensure consistency
- Only performs database writes when there are actual changes to make
- Batch inserts audit logs for efficiency
- Sorts final results in application layer, not database

### Idempotency
- No-op when all requested permissions are already assigned
- Efficient lookup using in-memory maps for duplicate detection
- Returns current state regardless of whether changes were made

## Response Format

### Success Response (200)
```json
{
  "meta": {
    "requestId": "req_..."
  },
  "data": [
    {
      "id": "perm_...",
      "name": "documents.read"
    }
  ]
}
```

### Error Response (4xx)
```json
{
  "meta": {
    "requestId": "req_..."
  },
  "error": {
    "code": "...",
    "message": "Human-readable error message"
  }
}
```

## Integration with Other Systems

### Cache Invalidation
- Key permissions cache is invalidated after successful operations
- Ensures immediate consistency for subsequent verification requests

### Role-Based Permissions
- Only affects direct permissions on the key
- Does not modify permissions granted through roles
- Response only includes direct permissions, not role-derived permissions