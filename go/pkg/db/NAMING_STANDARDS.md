# Database Query Naming Standards

This document establishes consistent naming conventions for SQL queries and files in the Unkey database layer to create hierarchical organization and improve maintainability.

## Overview

Our database queries use [sqlc](https://sqlc.dev/) to generate Go code from SQL. This naming strategy creates natural alphabetical groupings that make the codebase easier to navigate and understand.

## Benefits of Hierarchical Naming

- **Natural grouping** - All operations for an entity cluster together when sorted
- **Predictable file locations** - Developers can guess where to find queries
- **Easier navigation** - IDE file explorers show logical organization
- **Consistent generated code** - Go function names follow clear patterns
- **Reduced cognitive load** - Less time spent searching for existing queries

## File Naming Convention

Format: `{entity}_{operation}_{qualifier}.sql`

### Components

- **Entity**: Singular form of the primary table/entity (e.g., `api`, `key`, `permission`)
- **Operation**: Standardized operation name (see below)
- **Qualifier**: Optional descriptor for specific behavior or filtering

### Alphabetical Hierarchy Example

```
api_find_by_id.sql
api_insert.sql
api_soft_delete.sql
api_update_delete_protection.sql

identity_find_by_external_id.sql
identity_find_by_id.sql
identity_insert.sql
identity_list.sql
identity_list_ratelimits_by_id.sql
identity_soft_delete.sql
identity_update.sql

key_find_by_hash.sql
key_find_by_id.sql
key_find_for_verification.sql
key_insert.sql
key_list_by_key_auth_id.sql
key_soft_delete_many_by_key_auth_id.sql

permission_find_by_id.sql
permission_insert.sql
permission_list.sql
permission_list_by_key_id.sql
permission_list_direct_by_key_id.sql
```

## Standard Operations

### Single Record Operations
- `find_by_{field}` → `Find{Entity}By{Field}` - Single record lookup
- `insert` → `Insert{Entity}` - Create single record
- `update_{field}` → `Update{Entity}{Field}` - Modify specific field(s)
- `delete` → `Delete{Entity}` - Hard delete
- `soft_delete` → `SoftDelete{Entity}` - Soft delete (set deleted flag)

### Multiple Record Operations
- `list` → `List{Entity}` - General listing with pagination/filtering
- `list_by_{field}` → `List{Entity}By{Field}` - Multiple records by specific field
- `insert_many` → `InsertMany{Entity}` - Bulk insert operations
- `delete_many_by_{field}` → `DeleteMany{Entity}By{Field}` - Bulk delete operations
- `soft_delete_many_by_{field}` → `SoftDeleteMany{Entity}By{Field}` - Bulk soft delete

### Special Operations
- `list_{qualifier}` → `List{Entity}{Qualifier}` - Specialized listing (e.g., `list_matches`, `list_active`)

## Query Name Convention (within SQL files)

Format: `{Operation}{Entity}{Qualifier?}`

### Components
- **Operation**: PascalCase operation verb (Find, List, Insert, Update, Delete)
- **Entity**: PascalCase singular entity name
- **Qualifier**: Additional context in PascalCase

### Examples
```sql
-- name: FindApiById :one
-- name: ListKeysByKeyAuthId :many
-- name: InsertPermission :exec
-- name: ListRatelimitOverrideMatches :many
-- name: DeleteManyKeyPermissionsByPermissionId :exec
```

## Cross-Entity Query Guidelines

For queries spanning multiple tables, choose the primary entity based on:

1. **What's being returned** - Use the entity type of the primary return data
2. **What's being modified** - Use the entity being changed for mutations
3. **Business context** - Choose the most important entity from a business perspective

### Naming Patterns for Cross-Entity Queries

#### Association Table Operations
Use compound entity names for junction table operations:
```sql
-- File: key_permission_insert.sql
-- Query: InsertKeyPermission

-- File: key_permission_delete_by_key_and_permission_id.sql  
-- Query: DeleteKeyPermissionByKeyAndPermissionId

-- File: role_permission_list_by_role_id.sql
-- Query: ListRolePermissionsByRoleId
```

#### Joined Data Queries
Use the primary entity with descriptive qualifiers:
```sql
-- File: key_list_with_identity_by_key_auth_id.sql
-- Query: ListKeysWithIdentityByKeyAuthId

-- File: permission_list_direct_by_key_id.sql
-- Query: ListDirectPermissionsByKeyId
```

## Common Qualifiers

### Relationship Qualifiers
- `_with_{entity}` - Includes related entity data via JOIN
- `_direct` - Direct relationships only (no inherited/nested)
- `_inherited` - Includes inherited relationships

### Filtering Qualifiers  
- `_by_{field}_and_{field}` - Multiple filter conditions
- `_active` - Non-deleted/enabled records only
- `_expired` - Time-based filtering
- `_matches` - Pattern matching operations

### Scope Qualifiers
- `_many` - Indicates bulk operations
- `_all` - No filtering/pagination
- `_recent` - Time-limited results

## Migration Examples

### Before (Current State)
```sql
-- File: keys_find_by_key_auth_id.sql
-- name: FindKeysByKeyAuthId :many

-- File: permissions_by_key_id.sql
-- name: FindPermissionsForKey :many

-- File: identity_list_identities.sql  
-- name: ListIdentities :many

-- File: ratelimit_find_by_key_ids.sql
-- name: FindRatelimitsByKeyIds :many
```

### After (Following Standards)
```sql
-- File: key_list_by_key_auth_id.sql
-- name: ListKeysByKeyAuthId :many

-- File: permission_list_by_key_id.sql
-- name: ListPermissionsByKeyId :many

-- File: identity_list.sql
-- name: ListIdentities :many

-- File: ratelimit_list_by_key_ids.sql
-- name: ListRatelimitsByKeyIds :many
```

## Implementation Strategy

### Phase 1: Establish Standards
1. **Document current state** - Audit existing queries for patterns
2. **Plan migration batches** - Group related queries for batch updates
3. **Prepare tooling** - Create migration scripts and validation tools

### Phase 2: Gradual Migration
1. **Start with simple entities** - APIs, workspaces, single-table operations
2. **Handle complex cross-entity queries** - Permissions, keys with relations
3. **Update generated code references** - Fix all Go code after each batch

### Phase 3: Maintain Standards
1. **Code review enforcement** - Ensure new queries follow standards
2. **Documentation updates** - Keep examples current
3. **Tooling integration** - Add linting/validation to CI pipeline

## Validation Checklist

Before committing new queries, verify:

- [ ] File name follows `{entity}_{operation}_{qualifier}.sql` format
- [ ] Entity name is singular and matches primary table
- [ ] Operation is from the standard list
- [ ] Query name matches `{Operation}{Entity}{Qualifier}` format  
- [ ] Query name accurately describes the operation
- [ ] Multiple record operations use `List` prefix
- [ ] Bulk operations include `Many` in the name
- [ ] Cross-entity queries use appropriate primary entity
- [ ] File will sort alphabetically with related queries

## Examples by Entity Type

### Simple Entity (API)
```
api_find_by_id.sql              → FindApiById
api_insert.sql                  → InsertApi
api_soft_delete.sql             → SoftDeleteApi
api_update_delete_protection.sql → UpdateApiDeleteProtection
```

### Complex Entity (Key)
```
key_find_by_hash.sql                    → FindKeyByHash
key_find_by_id.sql                      → FindKeyById
key_find_for_verification.sql           → FindKeyForVerification
key_insert.sql                          → InsertKey
key_list_by_key_auth_id.sql            → ListKeysByKeyAuthId
key_soft_delete_many_by_key_auth_id.sql → SoftDeleteManyKeysByKeyAuthId
```

### Association Operations
```
key_permission_delete_by_key_and_permission_id.sql → DeleteKeyPermissionByKeyAndPermissionId
key_permission_insert.sql                          → InsertKeyPermission
key_role_insert.sql                                → InsertKeyRole
key_role_list_by_key_id.sql                       → ListKeyRolesByKeyId
```

## Questions and Edge Cases

### When Entity Choice is Unclear
Ask these questions:
1. What is the primary data being returned or modified?
2. Which entity would a developer most likely look for this operation under?
3. Which entity is most important from a business logic perspective?

### Complex Queries Spanning Many Tables
For very complex queries, choose the entity that represents the main "subject" of the query, even if other tables provide supporting data.

When in doubt, err on the side of being more descriptive rather than shorter. A slightly longer but clear name is better than a short but ambiguous one.

---

Following these standards will create a more maintainable, navigable, and consistent database layer that scales well as the application grows.