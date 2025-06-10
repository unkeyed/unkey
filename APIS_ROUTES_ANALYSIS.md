# APIs Routes Analysis: TypeScript v1 vs Go v2 Comparison

This document provides a detailed comparison of the APIs routes between the TypeScript v1 implementation and the Go v2 implementation, analyzing contracts, implementation differences, and identifying issues.

## Executive Summary

- **Migration Status**: 4/5 APIs routes migrated (80% complete)
- **Critical Gap**: `apis.deleteKeys` route is completely missing from Go v2
- **Method Changes**: All GET endpoints converted to POST (as expected)
- **Contract Issues**: Several implementation differences found that may break compatibility

## Route-by-Route Analysis

### ✅ `/v2/apis.createApi` - MIGRATED

**TypeScript v1**: `POST /v1/apis.createApi`  
**Go v2**: `POST /v2/apis.createApi`

#### Request Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| `name` | `string.min(3)` | `string` | ⚠️ **ISSUE** | Go missing minimum length validation |

#### Response Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| `apiId` | ✅ | ✅ | ✅ Match | - |
| `name` | ✅ Present | ❌ Missing | 🔴 **BREAKING** | Go doesn't return the name field |

#### Implementation Analysis
- **Authentication**: Both use root key auth ✅
- **Permissions**: Both check `api.*.create_api` permissions ✅
- **Database Operations**: Both create keyAuth and API records ✅
- **Audit Logging**: Both implement audit logs ✅
- **Transactions**: Both use database transactions ✅

#### Issues Found
1. 🔴 **BREAKING**: Go response missing `name` field that TypeScript returns
2. ⚠️ **VALIDATION**: Go missing minimum length validation for name field

---

### ⚠️ `/v2/apis.getApi` - MIGRATED WITH ISSUES

**TypeScript v1**: `GET /v1/apis.getApi?apiId={apiId}`  
**Go v2**: `POST /v2/apis.getApi`

#### Request Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| Method | `GET` | `POST` | ✅ Expected | Method change as per v2 standard |
| `apiId` | Query parameter | Body field | ✅ Expected | Input method change |

#### Response Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| `id` | ✅ | ✅ | ✅ Match | - |
| `workspaceId` | ✅ Present | ❌ Missing | 🔴 **BREAKING** | Critical field missing from Go |
| `name` | ✅ | ✅ | ✅ Match | - |

#### Implementation Analysis
- **Authentication**: Both use root key auth ✅
- **Permissions**: Both check appropriate read permissions ✅
- **Database Operations**: Both query API by ID ✅
- **Caching**: TypeScript uses cache, Go queries directly ⚠️ Performance impact
- **Error Handling**: Both handle not found cases ✅

#### Issues Found
1. 🔴 **BREAKING**: Go response missing `workspaceId` field
2. ⚠️ **PERFORMANCE**: Go implementation doesn't use caching like TypeScript does
3. ✅ **IMPROVEMENT**: Go has better error codes and fault handling

---

### ✅ `/v2/apis.deleteApi` - MIGRATED 

**TypeScript v1**: `POST /v1/apis.deleteApi`  
**Go v2**: `POST /v2/apis.deleteApi`

#### Request Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| `apiId` | ✅ | ✅ | ✅ Match | - |

#### Response Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| Empty response | `{}` | `{}` | ✅ Match | Both return empty response |

#### Implementation Analysis
- **Authentication**: Both use root key auth ✅
- **Permissions**: Both check delete permissions ✅
- **Delete Protection**: Both check delete protection ✅
- **Soft Delete**: Both implement soft deletion ✅
- **Cascade Delete**: Both delete associated keys ✅
- **Audit Logging**: Both implement comprehensive audit logs ✅
- **Cache Invalidation**: Both invalidate caches ✅

#### Issues Found
1. ⚠️ **BEHAVIORAL**: TypeScript deletes keys in same transaction, Go implementation appears to only soft-delete API
2. ✅ **IMPROVEMENT**: Go has better structured error handling

---

### ⚠️ `/v2/apis.listKeys` - MIGRATED WITH ISSUES

**TypeScript v1**: `GET /v1/apis.listKeys?apiId={apiId}&...`  
**Go v2**: `POST /v2/apis.listKeys`

#### Request Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| Method | `GET` | `POST` | ✅ Expected | Method change per v2 standard |
| `apiId` | Query param | Body field | ✅ Expected | Input method change |
| `limit` | Query param (1-100, default 100) | Body field | ✅ Match | - |
| `cursor` | Query param | Body field | ✅ Match | - |
| `externalId` | Query param | Body field | ✅ Match | - |
| `ownerId` | Query param (deprecated) | ❌ Missing | ⚠️ **COMPATIBILITY** | Deprecated field not supported |
| `decrypt` | Query param | Body field | ✅ Match | - |
| `revalidateKeysCache` | Query param | ❌ Missing | ⚠️ **FEATURE** | Cache control missing |

#### Response Contract Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| `keys` | Array of keys | Array of keys | ⚠️ **SCHEMA** | Different key schemas |
| `cursor` | String | String | ✅ Match | - |
| `total` | Number | ❌ Missing | 🔴 **BREAKING** | Total count missing from Go |

#### Key Schema Comparison
| Field | TypeScript v1 | Go v2 | Status | Notes |
|-------|---------------|-------|---------|-------|
| `id` | ✅ | `keyId` | ⚠️ **NAMING** | Field name changed |
| `start` | ✅ | ✅ | ✅ Match | - |
| `apiId` | ✅ Present | ❌ Missing | 🔴 **BREAKING** | API ID missing from Go |
| `workspaceId` | ✅ Present | ❌ Missing | 🔴 **BREAKING** | Workspace ID missing |
| `name` | ✅ | ✅ | ✅ Match | - |
| `ownerId` | ✅ Present | ❌ Missing | 🔴 **BREAKING** | Owner ID missing |
| `meta` | ✅ | ✅ | ✅ Match | - |
| `createdAt` | ✅ | ✅ | ✅ Match | - |
| `updatedAt` | ✅ | ✅ | ✅ Match | - |
| `expires` | ✅ | ✅ | ✅ Match | - |
| `ratelimit` | Complex object | ❌ Missing | 🔴 **BREAKING** | Rate limit info missing |
| `remaining` | ✅ Present | In `credits` object | ⚠️ **SCHEMA** | Moved to different structure |
| `refill` | Complex object | In `credits.refill` | ⚠️ **SCHEMA** | Restructured |
| `environment` | ✅ Present | ❌ Missing | 🔴 **BREAKING** | Environment missing |
| `plaintext` | ✅ | ✅ | ✅ Match | - |
| `roles` | ✅ | ✅ | ✅ Match | - |
| `permissions` | ✅ | ✅ | ✅ Match | - |
| `identity` | ✅ | ✅ | ✅ Match | - |

#### Implementation Analysis
- **Authentication**: Both use root key auth ✅
- **Permissions**: Both check read permissions ✅
- **Pagination**: Both implement cursor-based pagination ✅
- **Filtering**: Both support externalId filtering ✅
- **Decryption**: Both support key decryption ✅
- **Caching**: TypeScript has sophisticated caching, Go uses direct queries ⚠️

#### Issues Found
1. 🔴 **BREAKING**: Multiple missing fields in response (`total`, `apiId`, `workspaceId`, `ownerId`, `environment`)
2. 🔴 **BREAKING**: Missing `ratelimit` object structure
3. ⚠️ **SCHEMA**: Response structure significantly different
4. ⚠️ **FEATURE**: Missing cache revalidation option
5. ⚠️ **COMPATIBILITY**: Deprecated `ownerId` parameter not supported

---

### 🔴 `/v2/apis.deleteKeys` - MISSING

**TypeScript v1**: `POST /v1/apis.deleteKeys`  
**Go v2**: ❌ **NOT IMPLEMENTED**

#### Missing Functionality
- **Bulk key deletion** for an API
- **Permanent vs soft delete** options
- **Return count** of deleted keys

#### TypeScript Implementation Features
- Supports both permanent and soft deletion
- Bulk operation for performance
- Returns count of deleted keys
- Proper permission checks
- Transaction safety

#### Business Impact
- **HIGH**: No bulk key deletion capability in v2
- **OPERATIONAL**: Manual key deletion required
- **PERFORMANCE**: No efficient way to clean up API keys

---

## Summary of Issues

### 🔴 Critical Breaking Changes
1. **Missing `deleteKeys` route entirely**
2. **Response schema differences** in `getApi` and `listKeys`
3. **Missing fields** across multiple responses:
   - `workspaceId` in `getApi`
   - `total` count in `listKeys`
   - `apiId`, `ownerId`, `environment` in key objects
   - `ratelimit` structure in key objects

### ⚠️ Compatibility Concerns
1. **Method changes** from GET to POST (expected, but may break clients)
2. **Input parameter location changes** (query → body)
3. **Field naming inconsistencies** (`id` → `keyId`)
4. **Schema restructuring** (remaining/refill → credits structure)

### 🟡 Feature Gaps
1. **Cache control options** missing in `listKeys`
2. **Deprecated field support** missing (`ownerId`)
3. **Performance optimizations** (caching strategies)

### ✅ Improvements in Go Implementation
1. **Better error handling** with structured fault codes
2. **Improved type safety** with generated types
3. **Consistent service architecture**
4. **Better transaction management**

## Recommendations

### Immediate Actions Required
1. 🚨 **Implement `deleteKeys` route** - Critical missing functionality
2. 🚨 **Fix response schemas** to match TypeScript contracts
3. 🚨 **Add missing response fields** for backward compatibility

### Schema Fixes Needed
1. **Add `workspaceId`** to `getApi` response
2. **Add `total` count** to `listKeys` response  
3. **Add missing key fields**: `apiId`, `workspaceId`, `ownerId`, `environment`
4. **Restore `ratelimit` object** structure in key responses
5. **Consider field naming consistency** (`keyId` vs `id`)

### Feature Enhancements
1. **Implement caching strategy** similar to TypeScript version
2. **Add cache revalidation options**
3. **Support deprecated parameters** for transition period
4. **Add comprehensive validation** (e.g., name length minimums)

### Testing Requirements
1. **Contract testing** against TypeScript API responses
2. **Integration testing** with existing clients
3. **Performance testing** especially for `listKeys` without caching
4. **Migration testing** to ensure no data loss

## Migration Risk Assessment

### HIGH RISK ⚠️
- **`listKeys` schema changes** will break existing clients
- **Missing `deleteKeys`** blocks operational workflows
- **Missing fields** may cause client-side errors

### MEDIUM RISK 🟡
- **Method changes** require client updates
- **Performance differences** due to caching strategy changes

### LOW RISK ✅
- **Path versioning** clearly indicates breaking changes
- **Error handling improvements** enhance reliability

## Conclusion

The APIs routes migration is **80% functionally complete** but has **significant contract compatibility issues** that make it unsuitable for production use as a drop-in replacement. The missing `deleteKeys` route and breaking schema changes require immediate attention.

**Priority order for fixes:**
1. Implement missing `deleteKeys` route
2. Fix `listKeys` response schema to match TypeScript
3. Add missing fields to all responses
4. Implement caching strategy for performance
5. Add comprehensive testing for contract compatibility

The Go implementation shows good architectural improvements but needs contract alignment before it can serve as a reliable replacement for the TypeScript version.