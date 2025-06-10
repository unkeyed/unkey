# Test Scenarios for v2_keys_add_roles Endpoint

## Overview
This document outlines the test scenarios for the `v2/keys.addRoles` endpoint, which adds roles to an existing key. The operation is idempotent - adding roles that are already assigned has no effect.

## Success Scenarios (200 OK)

### Add Single Role by ID
- **Given**: Valid key ID and single role reference by ID
- **When**: POST /v2/keys.addRoles
- **Then**: Role is added to key, response includes all current roles
- **Audit**: AuthConnectRoleKeyEvent logged

### Add Single Role by Name
- **Given**: Valid key ID and single role reference by name
- **When**: POST /v2/keys.addRoles
- **Then**: Role is added to key, response includes all current roles
- **Audit**: AuthConnectRoleKeyEvent logged

### Add Multiple Roles (Mixed References)
- **Given**: Valid key ID and multiple roles (some by ID, some by name)
- **When**: POST /v2/keys.addRoles
- **Then**: All roles are added to key, response includes all current roles
- **Audit**: Multiple AuthConnectRoleKeyEvent entries logged

### Idempotent Behavior - Add Existing Roles
- **Given**: Key already has assigned roles, request includes some existing roles
- **When**: POST /v2/keys.addRoles
- **Then**: Only new roles are added, existing roles unchanged, no duplicate assignments
- **Audit**: Only new role additions logged

### Add Roles to Key with No Existing Roles
- **Given**: Key has no currently assigned roles
- **When**: POST /v2/keys.addRoles with multiple roles
- **Then**: All roles are added, response includes all added roles
- **Audit**: AuthConnectRoleKeyEvent for each role

### Add Roles to Key with Existing Roles
- **Given**: Key already has some assigned roles
- **When**: POST /v2/keys.addRoles with additional roles
- **Then**: New roles are added, existing roles preserved
- **Audit**: Only new role additions logged

## Bad Request Scenarios (400)

### Missing Key ID
- **Given**: Request body without keyId field
- **When**: POST /v2/keys.addRoles
- **Then**: 400 error - "keyId is required"

### Invalid Key ID Format
- **Given**: keyId that doesn't match expected format (e.g., not starting with 'key_')
- **When**: POST /v2/keys.addRoles
- **Then**: 400 error - "Invalid key ID format"

### Empty Roles Array
- **Given**: Valid keyId but empty roles array
- **When**: POST /v2/keys.addRoles
- **Then**: 400 error - "At least one role must be specified"

### Role Reference Missing ID and Name
- **Given**: Role reference object without id or name field
- **When**: POST /v2/keys.addRoles
- **Then**: 400 error - "Each role must specify either 'id' or 'name'"

### Invalid Role ID Format
- **Given**: Role reference with id that doesn't match expected format
- **When**: POST /v2/keys.addRoles
- **Then**: 400 error - "Invalid role ID format"

### Malformed Request Body
- **Given**: Invalid JSON or missing required fields
- **When**: POST /v2/keys.addRoles
- **Then**: 400 error with validation details

## Authentication Scenarios (401)

### Missing Authorization Header
- **Given**: Request without Authorization header
- **When**: POST /v2/keys.addRoles
- **Then**: 401 error - authentication required

### Invalid Root Key
- **Given**: Authorization header with invalid/non-existent root key
- **When**: POST /v2/keys.addRoles
- **Then**: 401 error - invalid credentials

### Expired Root Key
- **Given**: Authorization header with expired root key
- **When**: POST /v2/keys.addRoles
- **Then**: 401 error - token expired

## Authorization Scenarios (403)

### Insufficient Permissions - No UpdateKey
- **Given**: Valid root key without 'update_key' permission
- **When**: POST /v2/keys.addRoles
- **Then**: 403 error - insufficient permissions

### Root Key from Different Workspace
- **Given**: Valid root key from workspace A, trying to modify key in workspace B
- **When**: POST /v2/keys.addRoles
- **Then**: 403 error - unauthorized access

## Not Found Scenarios (404)

### Key Not Found
- **Given**: Non-existent key ID
- **When**: POST /v2/keys.addRoles
- **Then**: 404 error - "The specified key was not found"

### Key in Different Workspace
- **Given**: Valid key ID that exists but belongs to different workspace
- **When**: POST /v2/keys.addRoles
- **Then**: 404 error - "The specified key was not found"

### Role Not Found by ID
- **Given**: Valid key ID but non-existent role ID
- **When**: POST /v2/keys.addRoles
- **Then**: 404 error - "Role with ID 'role_xxx' was not found"

### Role Not Found by Name
- **Given**: Valid key ID but non-existent role name
- **When**: POST /v2/keys.addRoles
- **Then**: 404 error - "Role with name 'role_name' was not found"

### Role in Different Workspace
- **Given**: Valid key ID and role that exists but belongs to different workspace
- **When**: POST /v2/keys.addRoles
- **Then**: 404 error - "Role 'role_name' was not found"

## Edge Cases

### Role Reference with Both ID and Name
- **Given**: Role reference with both id and name fields
- **When**: POST /v2/keys.addRoles
- **Then**: ID takes precedence, role resolved by ID

### Duplicate Roles in Request
- **Given**: Same role referenced multiple times in request
- **When**: POST /v2/keys.addRoles
- **Then**: Role is added only once (deduplication)

### Mixed Success and Failure
- **Given**: Some valid roles and some invalid roles in same request
- **When**: POST /v2/keys.addRoles
- **Then**: First error encountered stops processing, transaction rolled back

## Response Format

### Success Response
```json
{
  "meta": {
    "requestId": "req_123"
  },
  "data": [
    {
      "id": "role_123",
      "name": "admin"
    },
    {
      "id": "role_456", 
      "name": "editor"
    }
  ]
}
```

### Error Response
```json
{
  "error": {
    "code": "DATA_KEY_NOT_FOUND",
    "message": "The specified key was not found.",
    "requestId": "req_123"
  }
}
```

## Audit Log Events

### Successful Role Addition
- **Event**: `authorization.connect_role_and_key`
- **Actor**: Root key used for request
- **Resources**: Key and Role involved
- **Display**: "Added role {role_name} to key {key_id}"

## Test Implementation Notes

1. **Idempotency**: Tests should verify that adding the same role multiple times has no effect
2. **Transaction Safety**: Failed operations should not leave partial changes
3. **Audit Logging**: All successful role additions should be logged
4. **Workspace Isolation**: Keys and roles from different workspaces should not be accessible
5. **Response Consistency**: Role list should be sorted alphabetically by name
6. **Permission Inheritance**: Tests should verify role-based permissions are properly inherited by the key