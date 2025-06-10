# Test Scenarios for v2_keys_set_permissions

## Overview

The `v2/keys.setPermissions` endpoint allows complete replacement of direct permission assignments for an API key using a differential update approach. Unlike incremental endpoints, this performs a wholesale replacement of all existing direct permission assignments.

## Endpoint Details

- **Method**: POST
- **Path**: `/v2/keys.setPermissions`
- **Permission Required**: `api.*.update_key`

## Success Scenarios (200 OK)

### Basic Permission Assignment

#### Set permissions using permission IDs
- **Given**: A key with permission1 assigned
- **When**: Setting permissions to [permission2, permission3] by ID
- **Then**: Key has only permission2 and permission3 (permission1 removed, permission2 and permission3 added)
- **Response**: Array of assigned permissions sorted alphabetically by name

#### Set permissions using permission slugs
- **Given**: A key with existing permissions
- **When**: Setting permissions using permission slugs instead of IDs
- **Then**: Permissions are resolved by slug and assigned correctly
- **Response**: Array showing final permission assignments

#### Mix permission IDs and slugs
- **Given**: A key with existing permissions
- **When**: Setting permissions using both IDs and slugs in the same request
- **Then**: All permissions are resolved correctly regardless of reference method
- **Response**: Consistent permission information with both ID and name

### Edge Cases

#### Set empty permissions (remove all)
- **Given**: A key with multiple permissions assigned
- **When**: Setting permissions to empty array `[]`
- **Then**: All existing direct permissions are removed from the key
- **Response**: Empty array indicating no direct permissions assigned

#### Set permissions with no changes
- **Given**: A key with permission1 assigned
- **When**: Setting permissions to [permission1] (same as current)
- **Then**: No database changes occur (idempotent operation)
- **Response**: Same permission configuration

#### Set multiple permissions with mixed changes
- **Given**: A key with [permission1, permission2] assigned
- **When**: Setting permissions to [permission1, permission3]
- **Then**: permission1 kept, permission2 removed, permission3 added
- **Response**: Final state [permission1, permission3]

#### Duplicate permissions in request
- **Given**: A valid key and permission
- **When**: Request includes the same permission multiple times
- **Then**: Permission is assigned only once (deduplication occurs)
- **Response**: Single instance of the permission

### Response Characteristics

#### Alphabetical sorting
- **Given**: Multiple permissions being assigned
- **When**: Any setPermissions operation
- **Then**: Response is always sorted alphabetically by permission name
- **Response**: Consistent ordering regardless of input order

#### Complete permission information
- **Given**: Successful permission assignment
- **When**: Any setPermissions operation
- **Then**: Response includes both permission ID and name for each permission
- **Response**: Full permission details for client convenience

## Error Scenarios

### 400 Bad Request

#### Missing required fields
- **Given**: Request missing `keyId` or `permissions`
- **When**: Submitting incomplete request
- **Then**: Schema validation error
- **Response**: BadRequestErrorResponse with validation details

#### Invalid keyId format
- **Given**: keyId with invalid format (too short, wrong prefix, etc.)
- **When**: Submitting request
- **Then**: Schema validation error
- **Response**: BadRequestErrorResponse

#### Permission with neither ID nor slug
- **Given**: Permission object with both `id` and `slug` empty/null
- **When**: Submitting request
- **Then**: Validation error with specific message
- **Response**: "Each permission must specify either 'id' or 'slug'"

#### Malformed JSON
- **Given**: Invalid JSON structure
- **When**: Submitting request
- **Then**: JSON parsing error
- **Response**: BadRequestErrorResponse with schema validation error

### 401 Unauthorized

#### Missing authorization header
- **Given**: Request without Authorization header
- **When**: Submitting request
- **Then**: Authentication error
- **Response**: UnauthorizedErrorResponse

#### Invalid authorization token
- **Given**: Request with non-existent or invalid root key
- **When**: Submitting request
- **Then**: Authentication error
- **Response**: UnauthorizedErrorResponse

#### Malformed authorization header
- **Given**: Authorization header without "Bearer " prefix
- **When**: Submitting request
- **Then**: Authentication error
- **Response**: BadRequestErrorResponse

### 403 Forbidden

#### Missing update_key permission
- **Given**: Root key without `api.*.update_key` permission
- **When**: Attempting to set permissions
- **Then**: Authorization error
- **Response**: ForbiddenErrorResponse

#### No permissions at all
- **Given**: Root key with no permissions
- **When**: Attempting to set permissions
- **Then**: Authorization error
- **Response**: ForbiddenErrorResponse

### 404 Not Found

#### Non-existent key ID
- **Given**: keyId that doesn't exist in any workspace
- **When**: Submitting request with valid permission
- **Then**: Key not found error
- **Response**: "The specified key was not found"
- **Status**: 404 with proper error structure

#### Non-existent permission ID
- **Given**: Valid key but completely non-existent permission ID
- **When**: Submitting request
- **Then**: Permission not found error with specific permission ID
- **Response**: "Permission with ID 'perm_nonexistent123' was not found"
- **Status**: 404 with proper error structure

#### Non-existent permission slug
- **Given**: Valid key but completely non-existent permission slug
- **When**: Submitting request
- **Then**: Permission not found error with specific permission slug
- **Response**: "Permission with slug 'nonexistent-permission' was not found"
- **Status**: 404 with proper error structure

#### Key from different workspace (isolation)
- **Given**: Root key authorized for workspace A, key exists in workspace B
- **When**: Attempting to set permissions
- **Then**: Key not found due to workspace isolation
- **Response**: "The specified key was not found"
- **Status**: 404 (key exists but not accessible)

#### Permission from different workspace (isolation)
- **Given**: Valid key in workspace A, permission exists in workspace B
- **When**: Attempting to set permissions by ID or slug
- **Then**: Permission not found due to workspace isolation
- **Response By ID**: "Permission 'permission_slug' was not found"
- **Response By Slug**: "Permission with slug 'permission_slug' was not found"
- **Status**: 404 (permission exists but not accessible)

#### Valid format but non-existent key
- **Given**: Properly formatted keyId (with correct prefix) that doesn't exist
- **When**: Submitting request
- **Then**: Key not found error
- **Response**: "The specified key was not found"
- **Status**: 404 (distinguishes format vs existence)

#### Valid format but non-existent permission
- **Given**: Properly formatted permissionId (with correct prefix) that doesn't exist
- **When**: Submitting request
- **Then**: Permission not found error
- **Response**: "Permission with ID 'perm_validformat123' was not found"
- **Status**: 404 (distinguishes format vs existence)

#### Multiple permissions with early failure
- **Given**: Request with multiple permissions where first one doesn't exist
- **When**: Submitting request
- **Then**: Fails on first non-existent permission (fail-fast behavior)
- **Response**: Error for the first problematic permission
- **Status**: 404 (doesn't process remaining permissions)

## Implementation Details

### Differential Update Algorithm
1. Fetch current direct permissions for the key
2. Resolve all requested permissions (by ID and name)
3. Calculate permissions to remove (current - requested)
4. Calculate permissions to add (requested - current)
5. Execute changes in transaction:
   - Remove obsolete permission assignments
   - Add new permission assignments
6. Audit log the changes
7. Return final state

### Database Operations
- Uses `FindDirectPermissionsForKey` to get current state
- Uses `FindPermissionById` and `FindPermissionBySlugAndWorkspace` for resolution
- Uses `DeleteKeyPermissionByKeyIdAndPermissionId` for selective removal
- Uses `InsertKeyPermission` for new assignments
- All operations wrapped in transaction for atomicity

### Audit Logging
- Uses typed event constants: `AuthConnectPermissionKeyEvent` for additions, `AuthDisconnectPermissionKeyEvent` for removals
- Creates separate audit log entry for each permission change (not a single bulk operation)
- Each entry includes both key and permission as resources for complete traceability
- Provides descriptive display messages for each individual change

### Security Features
- **Workspace Isolation**: Keys and permissions must belong to authorized workspace
- **Permission Validation**: Requires `api.*.update_key` permission
- **Granular Audit Logging**: Each permission addition/removal logged as separate entry with typed events
- **Atomic Operations**: Transaction ensures consistency

### Performance Considerations
- **Differential Updates**: Only changes what's necessary
- **Efficient Queries**: Minimal database operations
- **Transaction Scope**: Short-lived transactions
- **Response Sorting**: Client-friendly alphabetical ordering

## Business Rules

1. **Complete Replacement**: Operation replaces ALL existing direct permission assignments
2. **Workspace Isolation**: All resources must belong to the same workspace
3. **Permission Resolution**: Supports both ID and slug-based permission references
4. **Idempotent**: Setting the same permissions multiple times has no effect
5. **Atomic**: All changes succeed or all fail (no partial updates)
6. **Auditable**: Each permission change is logged individually with proper event types for detailed tracking
7. **Immediate Effect**: Changes take effect immediately for new verifications
8. **Direct Permissions Only**: This endpoint only affects direct permissions, not permissions granted through roles
9. **Role Permissions Preserved**: Permissions granted through roles are unaffected by this operation
10. **Permission Creation**: When using slugs with `create: true`, permissions can be created on-the-fly using the slug as both name and slug