# API Migration Analysis: TypeScript (v1) to Go (v2)

This document provides a comprehensive comparison between the TypeScript v1 API routes and the Go v2 API routes to track migration progress and identify differences.

## Overview

- **TypeScript API**: Located in `apps/api/src/routes/`, uses v1 endpoints with mixed HTTP methods
- **Go API**: Located in `go/apps/api/routes/`, uses v2 endpoints with POST-only methodology
- **Migration Status**: Partial migration completed, several routes still missing in Go implementation

## Key Differences

### 1. API Versioning
- **TypeScript**: Uses `v1` prefix (e.g., `/v1/apis.createApi`)
- **Go**: Uses `v2` prefix (e.g., `/v2/apis.createApi`)

### 2. HTTP Methods
- **TypeScript**: Mixed methods (GET for retrievals, POST for mutations)
- **Go**: POST-only approach for all endpoints

### 3. Route Structure
- **TypeScript**: Single file per route with tests
- **Go**: Directory per route with `handler.go` file

## Detailed Route Comparison

### ✅ APIs Routes

| TypeScript v1 | Go v2 | Status | Differences |
|---------------|-------|---------|-------------|
| `GET /v1/apis.getApi` | `POST /v2/apis.getApi` | ✅ Migrated | Method changed to POST |
| `POST /v1/apis.createApi` | `POST /v2/apis.createApi` | ✅ Migrated | Path version updated |
| `POST /v1/apis.deleteApi` | `POST /v2/apis.deleteApi` | ✅ Migrated | Path version updated |
| `GET /v1/apis.listKeys` | `POST /v2/apis.listKeys` | ✅ Migrated | Method changed to POST |
| `POST /v1/apis.deleteKeys` | ❌ Not implemented | 🔴 Missing | - |

### ✅ Identities Routes

| TypeScript v1 | Go v2 | Status | Differences |
|---------------|-------|---------|-------------|
| `POST /v1/identities.createIdentity` | `POST /v2/identities.createIdentity` | ✅ Migrated | Path version updated |
| `POST /v1/identities.deleteIdentity` | `POST /v2/identities.deleteIdentity` | ✅ Migrated | Path version updated |
| `GET /v1/identities.getIdentity` | `POST /v2/identities.getIdentity` | ✅ Migrated | Method changed to POST |
| `GET /v1/identities.listIdentities` | `POST /v2/identities.listIdentities` | ✅ Migrated | Method changed to POST |
| `POST /v1/identities.updateIdentity` | `POST /v2/identities.updateIdentity` | ✅ Migrated | Path version updated |

### ✅ Permissions Routes

| TypeScript v1 | Go v2 | Status | Differences |
|---------------|-------|---------|-------------|
| `POST /v1/permissions.createPermission` | `POST /v2/permissions.createPermission` | ✅ Migrated | Path version updated |
| `POST /v1/permissions.createRole` | `POST /v2/permissions.createRole` | ✅ Migrated | Path version updated |
| `POST /v1/permissions.deletePermission` | `POST /v2/permissions.deletePermission` | ✅ Migrated | Path version updated |
| `POST /v1/permissions.deleteRole` | `POST /v2/permissions.deleteRole` | ✅ Migrated | Path version updated |
| `GET /v1/permissions.getPermission` | `POST /v2/permissions.getPermission` | ✅ Migrated | Method changed to POST |
| `GET /v1/permissions.getRole` | `POST /v2/permissions.getRole` | ✅ Migrated | Method changed to POST |
| `GET /v1/permissions.listPermissions` | `POST /v2/permissions.listPermissions` | ✅ Migrated | Method changed to POST |
| `GET /v1/permissions.listRoles` | `POST /v2/permissions.listRoles` | ✅ Migrated | Method changed to POST |

### ⚠️ Ratelimits Routes

| TypeScript v1 | Go v2 | Status | Differences |
|---------------|-------|---------|-------------|
| `POST /v1/ratelimits.deleteOverride` | `POST /v2/ratelimit.deleteOverride` | ✅ Migrated | Path name changed (ratelimits → ratelimit) |
| `GET /v1/ratelimits.getOverride` | `POST /v2/ratelimit.getOverride` | ✅ Migrated | Method changed to POST, path name changed |
| `POST /v1/ratelimits.limit` | `POST /v2/ratelimit.limit` | ✅ Migrated | Path name changed (ratelimits → ratelimit) |
| `GET /v1/ratelimits.listOverrides` | `POST /v2/ratelimit.listOverrides` | ✅ Migrated | Method changed to POST, path name changed |
| `POST /v1/ratelimits.setOverride` | `POST /v2/ratelimit.setOverride` | ✅ Migrated | Path name changed (ratelimits → ratelimit) |

### 🔴 Keys Routes (NOT MIGRATED)

| TypeScript v1 | Go v2 | Status | Priority |
|---------------|-------|---------|----------|
| `POST /v1/keys.addPermissions` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.addRoles` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.createKey` | ❌ Not implemented | 🔴 Missing | **Critical** |
| `POST /v1/keys.deleteKey` | ❌ Not implemented | 🔴 Missing | High |
| `GET /v1/keys.getKey` | ❌ Not implemented | 🔴 Missing | High |
| `GET /v1/keys.getVerifications` | ❌ Not implemented | 🔴 Missing | Medium |
| `POST /v1/keys.removePermissions` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.removeRoles` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.setPermissions` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.setRoles` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.updateKey` | ❌ Not implemented | 🔴 Missing | High |
| `POST /v1/keys.updateRemaining` | ❌ Not implemented | 🔴 Missing | Medium |
| `POST /v1/keys.verifyKey` | ❌ Not implemented | 🔴 Missing | **Critical** |
| `POST /v1/keys.whoami` | ❌ Not implemented | 🔴 Missing | Medium |

### 🔴 Analytics Routes (NOT MIGRATED)

| TypeScript v1 | Go v2 | Status | Priority |
|---------------|-------|---------|----------|
| `GET /v1/analytics.getVerifications` | ❌ Not implemented | 🔴 Missing | Medium |

### 🔴 Migrations Routes (NOT MIGRATED)

| TypeScript v1 | Go v2 | Status | Priority |
|---------------|-------|---------|----------|
| `POST /v1/migrations.createKey` | ❌ Not implemented | 🔴 Missing | Low |
| `POST /v1/migrations.enqueueKeys` | ❌ Not implemented | 🔴 Missing | Low |

### ✅ Utility Routes

| TypeScript v1 | Go v2 | Status | Differences |
|---------------|-------|---------|-------------|
| `GET /v1/liveness` | `POST /v2/liveness` | ✅ Migrated | Method changed to POST |

### 🔴 Legacy Routes (NOT MIGRATED)

| TypeScript v1 | Go v2 | Status | Notes |
|---------------|-------|---------|-------|
| `GET /v1/apis/{apiId}/keys` (legacy) | ❌ Not implemented | 🔴 Missing | Legacy route, may not need migration |
| `POST /v1/keys` (legacy) | ❌ Not implemented | 🔴 Missing | Legacy route, may not need migration |
| `POST /v1/keys/verify` (legacy) | ❌ Not implemented | 🔴 Missing | Legacy route, may not need migration |

## Migration Progress Summary

### ✅ Completed (6/11 categories)
- **APIs**: 4/5 routes migrated (80%)
- **Identities**: 5/5 routes migrated (100%)
- **Permissions**: 8/8 routes migrated (100%)
- **Ratelimits**: 5/5 routes migrated (100%)
- **Liveness**: 1/1 routes migrated (100%)
- **Reference**: New route in Go

### 🔴 Missing (5/11 categories)
- **Keys**: 0/14 routes migrated (0%) - **CRITICAL PRIORITY**
- **Analytics**: 0/1 routes migrated (0%)
- **Migrations**: 0/2 routes migrated (0%)
- **APIs**: 1/5 routes missing (`deleteKeys`)
- **Legacy**: 0/3 routes migrated (likely intentional)

### Overall Migration Status
- **Total Routes in TS**: ~45 routes
- **Total Routes Migrated**: ~25 routes
- **Migration Progress**: ~56%

## Critical Missing Functionality

### 🚨 High Priority (Core API Functionality)
1. **`/v1/keys.createKey`** - Core functionality for creating API keys
2. **`/v1/keys.verifyKey`** - Core functionality for key verification
3. **`/v1/keys.deleteKey`** - Essential key management
4. **`/v1/keys.getKey`** - Key retrieval functionality
5. **Key permissions/roles management** (addPermissions, removePermissions, setPermissions, addRoles, removeRoles, setRoles)
6. **`/v1/keys.updateKey`** - Key modification functionality

### ⚠️ Medium Priority
1. **`/v1/analytics.getVerifications`** - Analytics functionality
2. **`/v1/keys.getVerifications`** - Key usage analytics
3. **`/v1/keys.updateRemaining`** - Usage limit management
4. **`/v1/keys.whoami`** - Key information endpoint

### 📝 Low Priority
1. **Migration routes** - Likely used for one-time migrations
2. **Legacy routes** - May be intentionally deprecated

## Recommendations

### Immediate Actions Required
1. **Prioritize Keys routes migration** - This is the core functionality of the API
2. **Implement `createKey` and `verifyKey` first** - These are absolutely critical
3. **Add missing `deleteKeys` route in APIs** - Complete the APIs migration
4. **Review path naming consistency** - Consider whether `ratelimits` vs `ratelimit` is intentional

### Migration Strategy
1. **Phase 1**: Complete Keys routes (critical for core functionality)
2. **Phase 2**: Add Analytics routes (important for monitoring)
3. **Phase 3**: Add remaining utility routes
4. **Phase 4**: Evaluate need for Legacy and Migration routes

### Technical Considerations
1. **Method Changes**: Ensure all clients can handle GET → POST method changes
2. **Path Changes**: Update any hardcoded paths that reference `ratelimits` vs `ratelimit`
3. **Authentication**: Verify that POST-only approach works with existing auth mechanisms
4. **Testing**: Ensure comprehensive test coverage for migrated routes

## Notes
- This analysis is based on file structure and route registration patterns
- Actual implementation details may vary and should be verified by examining individual handler files
- Legacy routes may be intentionally not migrated as part of API cleanup
- The POST-only approach in v2 appears to be a deliberate architectural decision