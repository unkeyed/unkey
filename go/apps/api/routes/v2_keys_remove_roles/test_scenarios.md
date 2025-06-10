# Test Scenarios for v2_keys_remove_roles

This document outlines the comprehensive test scenarios for the `/v2/keys.removeRoles` endpoint.

## Overview

The endpoint allows removing roles directly from an API key. It's an idempotent operation that only removes roles that are currently assigned to the key. Unlike the removePermissions endpoint, this returns the remaining roles still assigned to the key after the removal operation.

## Test Coverage

### 200 - Success Scenarios

#### Basic Operations
- **Remove single role by ID**: Verifies that a single role can be removed using its role ID
- **Remove single role by name**: Verifies that a single role can be removed using its role name
- **Remove multiple roles**: Verifies that multiple roles can be removed in a single request using mixed ID and name references

#### Edge Cases
- **Idempotent operation**: Verifies that removing a role that isn't assigned is a no-op (doesn't cause an error)
- **Partial removal**: Verifies that only specified roles are removed, leaving others intact
- **Remove all roles**: Verifies that all direct roles can be removed from a key
- **Remaining roles response**: Verifies that the endpoint returns the remaining roles after removal

#### Verification Points
- Response status is 200
- Response includes `requestId` in metadata
- Response data contains array of remaining roles with `id` and `name` fields
- Database state reflects the removals
- Audit logs are created for role removals
- Only roles that were actually assigned get removed (idempotent behavior)
- Roles not in the removal request remain unchanged
- Response roles are sorted alphabetically by name for consistency

### 400 - Validation Errors

#### Request Validation
- **Missing keyId**: Request with empty or missing key ID
- **Empty keyId**: Request with empty key ID string
- **Missing roles**: Request with no roles specified
- **Empty roles array**: Request with empty roles array
- **Invalid keyId format**: Request with malformed key ID (not starting with 'key_')

#### Data Validation
- **Role missing both id and name**: Role object with neither `id` nor `name` specified
- **Role not found by id**: Request references a role ID that doesn't exist
- **Role not found by name**: Request references a role name that doesn't exist
- **Key not found**: Request references a key ID that doesn't exist
- **Invalid role ID format**: Role ID that doesn't start with 'role_'

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
- **Role belongs to different workspace**: Attempting to remove a role from a different workspace

### 404 - Not Found Errors

#### Resource Not Found
- **Key not found**: Specified key ID doesn't exist in the authorized workspace
- **Role not found by ID**: Specified role ID doesn't exist in the authorized workspace
- **Role not found by name**: Specified role name doesn't exist in the authorized workspace

#### Cross-Workspace Access
- **Role from different workspace by ID**: Role exists but belongs to a different workspace (returns 404, not 403, for security)
- **Key from different workspace**: Key exists but belongs to a different workspace (returns 404, not 403, for security)

## Security Considerations

### Workspace Isolation
- Keys and roles are strictly isolated by workspace
- Cross-workspace access attempts return 404 (not 403) to avoid information disclosure
- Root keys can only operate on resources within their authorized workspace

### Permission Requirements
- Requires `api.*.update_key` permission on the root key
- Wildcard permissions (`*`) are supported for broader access
- Permission hierarchy is enforced (specific permissions don't grant broader access)

### Audit Trail
- All role removals are logged to the audit system with `auth.disconnect_role_key` event
- Includes actor information (root key), target resources, and operation details
- Failed operations are not logged (only successful removals)
- No-op operations (removing non-assigned roles) don't generate audit logs

## Performance Considerations

### Database Operations
- Uses transactions to ensure consistency
- Only performs database writes when there are actual changes to make
- Batch inserts audit logs for efficiency
- Efficient lookup using in-memory maps for role validation
- Uses `DeleteKeyRoleByKeyIdAndRoleId` for selective removal

### Idempotency
- No-op when attempting to remove roles that aren't assigned
- Efficient checking of current roles before attempting removal
- Returns success regardless of whether changes were made

## Response Format

### Success Response (200)
```json
{
  "meta": {
    "requestId": "req_..."
  },
  "data": [
    {
      "id": "role_abc123",
      "name": "developer"
    }
  ]
}
```

Note: Unlike `removePermissions`, this endpoint returns the remaining roles array, similar to `addRoles`. An empty array indicates the key has no roles assigned.

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
- Key roles cache is invalidated after successful operations
- Ensures immediate consistency for subsequent verification requests

### Permission-Based Access
- Only affects direct role assignments on the key
- Does not modify permissions granted directly to the key
- Removing roles may reduce effective permissions for the key

## Differences from Other Role Endpoints

### Response Format
- `removeRoles` returns the remaining roles array
- `addRoles` returns all roles (including newly added ones)
- `setRoles` returns the complete replacement set

### Operation Behavior
- `removeRoles` is purely subtractive
- `addRoles` is purely additive
- `setRoles` is a complete replacement operation
- All are idempotent operations

### Use Cases
- `removeRoles` is ideal for revoking specific role-based access
- Combined with `addRoles`, enables fine-grained role management
- For wholesale replacement, use `setRoles` instead

## Testing Strategy

### Unit Testing Focus
- Idempotent behavior with non-assigned roles
- Partial removal scenarios (removing some but not all roles)
- Remaining roles response validation and sorting
- Audit log generation for actual removals only
- Role resolution by both ID and name

### Integration Testing Focus
- Cross-workspace isolation
- Transaction rollback on errors
- Cache invalidation timing
- Audit log persistence and formatting
- Database consistency checks

## Database Queries Used

### Read Operations
- `FindKeyByID` - Validates key existence and workspace ownership
- `FindRolesForKey` - Gets current role assignments for the key
- `FindRoleById` - Resolves role references by ID
- `FindRoleByNameAndWorkspace` - Resolves role references by name

### Write Operations
- `DeleteKeyRoleByKeyIdAndRoleId` - Removes specific role assignments
- Audit log insertion for tracking changes

## Error Scenarios

### Database Errors
- Connection failures during role lookup
- Transaction failures during role removal
- Audit log insertion failures

### Validation Errors
- Malformed request payloads
- Invalid role references (both ID and name)
- Invalid key references

### Business Logic Errors
- Cross-workspace access attempts
- Insufficient permissions for the operation
- Role not found scenarios

## Performance Benchmarks

### Expected Performance
- Single role removal: < 100ms
- Multiple role removal (5-10 roles): < 200ms
- Large batch removal (50+ roles): < 500ms

### Optimization Strategies
- Efficient role lookup using maps
- Batch audit log insertion
- Minimal database round trips
- Transaction scope optimization