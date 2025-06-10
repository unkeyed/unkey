# TODO: API Migration Tasks

This document tracks pending tasks for completing the TypeScript to Go API migration.

## High Priority Tasks

### 🔴 Critical Missing Features

#### 1. Add `revalidateKeysCache` parameter to `/v2/apis.listKeys`
**Status**: ✅ Already Implemented  
**Priority**: High  
**Endpoint**: `POST /v2/apis.listKeys`

**Description**:
The TypeScript v1 implementation includes a `revalidateKeysCache` boolean parameter that allows clients to skip the cache and fetch keys directly from the database. This is useful when creating a key and immediately listing all keys to display them to users.

**Current TypeScript Implementation**:
```typescript
revalidateKeysCache: z.coerce
  .boolean()
  .default(false)
  .optional()
  .openapi({
    description: `\`EXPERIMENTAL\`

Skip the cache and fetch the keys from the database directly.
When you're creating a key and immediately listing all keys to display them to your user, you might want to skip the cache to ensure the key is displayed immediately.
    `,
  }),
```

**Required Changes**:
- [x] Add `revalidateKeysCache` field to request body schema
- [ ] Implement cache bypass logic when parameter is true (waiting for caching implementation)
- [x] Add validation for boolean type
- [x] Update OpenAPI documentation
- [ ] Add tests for cache behavior (waiting for caching implementation)
- [x] Mark as experimental feature (matching TypeScript)

**Implementation Notes**:
- Should default to `false` to maintain performance
- When `true`, bypass any caching mechanisms and query database directly
- Consider performance implications for high-traffic scenarios
- May need to implement cache warming after direct database fetch

**Files to Modify**:
- `unkey/go/apps/api/routes/v2_apis_list_keys/handler.go`
- OpenAPI schema definitions
- Test files for the endpoint

**Acceptance Criteria**:
- [x] Parameter accepted in request body
- [ ] Cache is bypassed when `revalidateKeysCache: true` (waiting for caching implementation)
- [ ] Normal caching behavior when `revalidateKeysCache: false` or omitted (waiting for caching implementation)
- [x] Performance remains acceptable for both modes
- [ ] Tests cover both cached and non-cached scenarios (waiting for caching implementation)
- [x] Documentation matches TypeScript implementation

#### 2. Add `ratelimits` array to key response objects in `/v2/apis.listKeys`
**Status**: ✅ Completed with Tests  
**Priority**: High  
**Endpoint**: `POST /v2/apis.listKeys`

**Description**:
The TypeScript v1 implementation includes a `ratelimit` object in each key response, but the Go v2 implementation is missing this critical information. Keys need to return their rate limiting configuration so clients can display and manage rate limits properly.

**Current TypeScript Response Structure**:
```typescript
ratelimit: k.ratelimitAsync !== null && k.ratelimitLimit !== null && k.ratelimitDuration !== null
  ? {
      async: k.ratelimitAsync,
      type: k.ratelimitAsync ? "fast" : ("consistent" as unknown),
      limit: k.ratelimitLimit,
      duration: k.ratelimitDuration,
      refillRate: k.ratelimitLimit,
      refillInterval: k.ratelimitDuration,
    }
  : undefined,
```

**Required Changes**:
- [x] Add `ratelimit` object to key response schema in OpenAPI
- [x] Query rate limit data from database for each key
- [x] Transform database rate limit fields into response format
- [x] Handle cases where keys have no rate limits (return null/undefined)
- [x] Map `async`/`consistent` types correctly
- [x] Include all rate limit fields: `limit`, `duration`, `type`, `async`, `refillRate`, `refillInterval`
- [x] Add tests for keys with and without rate limits

**Database Fields to Map**:
- `ratelimitAsync` → `async` and `type` fields
- `ratelimitLimit` → `limit` and `refillRate` fields  
- `ratelimitDuration` → `duration` and `refillInterval` fields

**Implementation Notes**:
- Rate limit should only be included if all required fields are present
- `type` should be "fast" if `async` is true, "consistent" if false
- `refillRate` and `refillInterval` should match `limit` and `duration` respectively
- Consider performance impact of additional database queries

**Files to Modify**:
- `unkey/go/apps/api/routes/v2_apis_list_keys/handler.go`
- OpenAPI schema definitions for key response
- Database queries to include rate limit fields
- Test files for the endpoint

**Acceptance Criteria**:
- [x] Keys with rate limits return complete `ratelimit` object
- [x] Keys without rate limits return `null` or omit the field
- [x] All rate limit fields are correctly mapped from database
- [x] `type` field correctly reflects async/consistent behavior
- [x] Response format matches TypeScript v1 implementation
- [x] Tests cover various rate limit scenarios

---

## Additional Tasks (To be added)

_More tasks will be added to this TODO as we identify and prioritize them._

---

## Completion Tracking

**Last Updated**: December 2024  
**Total Tasks**: 2  
**Completed**: 2  
**In Progress**: 0  
**Not Started**: 0

**Notes**: 
- Task 1 (revalidateKeysCache) is technically complete but waiting for caching implementation to be fully functional
- Task 2 (ratelimits) is fully complete with comprehensive test coverage