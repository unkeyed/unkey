# Test Scenarios for v2_keys_set_roles

## Overview

The `v2/keys.setRoles` endpoint allows complete replacement of role assignments for an API key using a differential update approach. Unlike incremental endpoints, this performs a wholesale replacement of all existing role assignments.

## Endpoint Details

- **Method**: POST
- **Path**: `/v2/keys.setRoles`
- **Permission Required**: `api.*.update_key`

## Success Scenarios (200 OK)

### Basic Role Assignment

#### Set roles using role IDs
- **Given**: A key with role1 assigned
- **When**: Setting roles to [role2, role3] by ID
- **Then**: Key has only role2 and role3 (role1 removed, role2 and role3 added)
- **Response**: Array of assigned roles sorted alphabetically by name

#### Set roles using role names
- **Given**: A key with existing roles
- **When**: Setting roles using role names instead of IDs
- **Then**: Roles are resolved by name and assigned correctly
- **Response**: Array showing final role assignments

#### Mix role IDs and names
- **Given**: A key with existing roles
- **When**: Setting roles using both IDs and names in the same request
- **Then**: All roles are resolved correctly regardless of reference method
- **Response**: Consistent role information with both ID and name

### Edge Cases

#### Set empty roles (remove all)
- **Given**: A key with multiple roles assigned
- **When**: Setting roles to empty array `[]`
- **Then**: All existing roles are removed from the key
- **Response**: Empty array indicating no roles assigned

#### Set roles with no changes
- **Given**: A key with role1 assigned
- **When**: Setting roles to [role1] (same as current)
- **Then**: No database changes occur (idempotent operation)
- **Response**: Same role configuration

#### Set multiple roles with mixed changes
- **Given**: A key with [role1, role2] assigned
- **When**: Setting roles to [role1, role3]
- **Then**: role1 kept, role2 removed, role3 added
- **Response**: Final state [role1, role3]

#### Duplicate roles in request
- **Given**: A valid key and role
- **When**: Request includes the same role multiple times
- **Then**: Role is assigned only once (deduplication occurs)
- **Response**: Single instance of the role

### Response Characteristics

#### Alphabetical sorting
- **Given**: Multiple roles being assigned
- **When**: Any setRoles operation
- **Then**: Response is always sorted alphabetically by role name
- **Response**: Consistent ordering regardless of input order

#### Complete role information
- **Given**: Successful role assignment
- **When**: Any setRoles operation
- **Then**: Response includes both role ID and name for each role
- **Response**: Full role details for client convenience

## Error Scenarios

### 400 Bad Request

#### Missing required fields
- **Given**: Request missing `keyId` or `roles`
- **When**: Submitting incomplete request
- **Then**: Schema validation error
- **Response**: BadRequestErrorResponse with validation details

#### Invalid keyId format
- **Given**: keyId with invalid format (too short, wrong prefix, etc.)
- **When**: Submitting request
- **Then**: Schema validation error
- **Response**: BadRequestErrorResponse

#### Role with neither ID nor name
- **Given**: Role object with both `id` and `name` empty/null
- **When**: Submitting request
- **Then**: Validation error with specific message
- **Response**: "Each role must specify either 'id' or 'name'"

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
- **When**: Attempting to set roles
- **Then**: Authorization error
- **Response**: ForbiddenErrorResponse

#### No permissions at all
- **Given**: Root key with no permissions
- **When**: Attempting to set roles
- **Then**: Authorization error
- **Response**: ForbiddenErrorResponse

### 404 Not Found

#### Non-existent key ID
- **Given**: keyId that doesn't exist in any workspace
- **When**: Submitting request with valid role
- **Then**: Key not found error
- **Response**: "The specified key was not found"
- **Status**: 404 with proper error structure

#### Non-existent role ID
- **Given**: Valid key but completely non-existent role ID
- **When**: Submitting request
- **Then**: Role not found error with specific role ID
- **Response**: "Role with ID 'role_nonexistent123' was not found"
- **Status**: 404 with proper error structure

#### Non-existent role name
- **Given**: Valid key but completely non-existent role name
- **When**: Submitting request
- **Then**: Role not found error with specific role name
- **Response**: "Role with name 'nonexistent-role' was not found"
- **Status**: 404 with proper error structure

#### Key from different workspace (isolation)
- **Given**: Root key authorized for workspace A, key exists in workspace B
- **When**: Attempting to set roles
- **Then**: Key not found due to workspace isolation
- **Response**: "The specified key was not found"
- **Status**: 404 (key exists but not accessible)

#### Role from different workspace (isolation)
- **Given**: Valid key in workspace A, role exists in workspace B
- **When**: Attempting to set roles by ID or name
- **Then**: Role not found due to workspace isolation
- **Response By ID**: "Role 'role_name' was not found"
- **Response By Name**: "Role with name 'role_name' was not found"
- **Status**: 404 (role exists but not accessible)

#### Valid format but non-existent key
- **Given**: Properly formatted keyId (with correct prefix) that doesn't exist
- **When**: Submitting request
- **Then**: Key not found error
- **Response**: "The specified key was not found"
- **Status**: 404 (distinguishes format vs existence)

#### Valid format but non-existent role
- **Given**: Properly formatted roleId (with correct prefix) that doesn't exist
- **When**: Submitting request
- **Then**: Role not found error
- **Response**: "Role with ID 'role_validformat123' was not found"
- **Status**: 404 (distinguishes format vs existence)

#### Multiple roles with early failure
- **Given**: Request with multiple roles where first one doesn't exist
- **When**: Submitting request
- **Then**: Fails on first non-existent role (fail-fast behavior)
- **Response**: Error for the first problematic role
- **Status**: 404 (doesn't process remaining roles)

## Implementation Details

### Differential Update Algorithm
1. Fetch current roles for the key
2. Resolve all requested roles (by ID and name)
3. Calculate roles to remove (current - requested)
4. Calculate roles to add (requested - current)
5. Execute changes in transaction:
   - Remove obsolete role assignments
   - Add new role assignments
6. Audit log the changes
7. Return final state

### Database Operations
- Uses `FindRolesForKey` to get current state
- Uses `FindRoleById` and `FindRoleByNameAndWorkspace` for resolution
- Uses `DeleteKeyRoleByKeyIdAndRoleId` for selective removal
- Uses `InsertKeyRole` for new assignments
- All operations wrapped in transaction for atomicity

### Audit Logging
- Uses typed event constants: `AuthConnectRoleKeyEvent` for additions, `AuthDisconnectRoleKeyEvent` for removals
- Creates separate audit log entry for each role change (not a single bulk operation)
- Each entry includes both key and role as resources for complete traceability
- Provides descriptive display messages for each individual change

### Security Features
- **Workspace Isolation**: Keys and roles must belong to authorized workspace
- **Permission Validation**: Requires `api.*.update_key` permission
- **Granular Audit Logging**: Each role addition/removal logged as separate entry with typed events
- **Atomic Operations**: Transaction ensures consistency

### Performance Considerations
- **Differential Updates**: Only changes what's necessary
- **Efficient Queries**: Minimal database operations
- **Transaction Scope**: Short-lived transactions
- **Response Sorting**: Client-friendly alphabetical ordering

## Business Rules

1. **Complete Replacement**: Operation replaces ALL existing role assignments
2. **Workspace Isolation**: All resources must belong to the same workspace
3. **Role Resolution**: Supports both ID and name-based role references
4. **Idempotent**: Setting the same roles multiple times has no effect
5. **Atomic**: All changes succeed or all fail (no partial updates)
6. **Auditable**: Each role change is logged individually with proper event types for detailed tracking
7. **Immediate Effect**: Changes take effect immediately for new verifications