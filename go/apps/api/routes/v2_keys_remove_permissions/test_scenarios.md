# Test Scenarios for v2_keys_remove_permissions

This document outlines the comprehensive test scenarios for the `/v2/keys.removePermissions` endpoint.

## Overview

The endpoint allows removing permissions directly from an API key. It's an idempotent operation that only removes permissions that are currently assigned to the key. Unlike the addPermissions endpoint, this returns an empty response object.

## Test Coverage

### 200 - Success Scenarios

#### Basic Operations
- **Remove single permission by ID**: Verifies that a single permission can be removed using its permission ID
- **Remove single permission by name**: Verifies that a single permission can be removed using its permission name
- **Remove multiple permissions**: Verifies that multiple permissions can be removed in a single request using mixed ID and name references

#### Edge Cases
- **Idempotent operation**: Verifies that removing a permission that isn't assigned is a no-op (doesn't cause an error)
- **Partial removal**: Verifies that only specified permissions are removed, leaving others intact
- **Remove all permissions**: Verifies that all direct permissions can be removed from a key
- **Empty response**: Verifies that the endpoint returns an empty object as specified in the OpenAPI schema

#### Verification Points
- Response status is 200
- Response includes `requestId` in metadata
- Response data is an empty object (not an array like addPermissions)
- Database state reflects the removals
- Audit logs are created for permission removals
- Only permissions that were actually assigned get removed (idempotent behavior)
- Permissions not in the removal request remain unchanged

### 400 - Validation Errors

#### Request Validation
- **Missing keyId**: Request with empty or missing key ID
- **Empty keyId**: Request with empty key ID string
- **Missing permissions**: Request with no permissions specified
- **Empty permissions array**: Request with empty permissions array
- **Invalid keyId format**: Request with malformed key ID

#### Data Validation
- **Permission missing both id and name**: Permission object with neither `id` nor `name` specified
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
- **Permission belongs to different workspace**: Attempting to remove a permission from a different workspace

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
- All permission removals are logged to the audit system
- Includes actor information (root key), target resources, and operation details
- Failed operations are not logged (only successful removals)
- No-op operations (removing non-assigned permissions) don't generate audit logs

## Performance Considerations

### Database Operations
- Uses transactions to ensure consistency
- Only performs database writes when there are actual changes to make
- Batch inserts audit logs for efficiency
- Efficient lookup using in-memory maps for permission validation

### Idempotency
- No-op when attempting to remove permissions that aren't assigned
- Efficient checking of current permissions before attempting removal
- Returns success regardless of whether changes were made

## Response Format

### Success Response (200)
```json
{
  "meta": {
    "requestId": "req_..."
  },
  "data": {}
}
```

Note: Unlike `addPermissions`, this endpoint returns an empty object, not the current permissions list. To get the updated permissions, use the `keys.getKey` endpoint.

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
- Removing all direct permissions doesn't affect role-derived permissions

## Differences from addPermissions

### Response Format
- `removePermissions` returns an empty object
- `addPermissions` returns the current direct permissions list

### Operation Behavior
- `removePermissions` is purely subtractive
- `addPermissions` is purely additive
- Both are idempotent operations

### Use Cases
- `removePermissions` is ideal for revoking specific access
- Combined with `addPermissions`, enables fine-grained permission management
- For wholesale replacement, use `setPermissions` instead

## Testing Strategy

### Unit Testing Focus
- Idempotent behavior with non-assigned permissions
- Partial removal scenarios (removing some but not all permissions)
- Empty response object validation
- Audit log generation for actual removals only

### Integration Testing Focus
- Cross-workspace isolation
- Transaction rollback on errors
- Cache invalidation timing
- Audit log persistence and formatting