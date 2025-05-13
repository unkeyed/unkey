# OpenAPI Migration Plan: TypeScript API to Go API

This document outlines the plan for migrating our OpenAPI specifications from the old TypeScript API to the new Go API with v2 endpoints.

## Migration Approach

1. Identify all existing routes in the TypeScript API
2. Map each route to a corresponding v2 route
3. Update the OpenAPI schema in `unkey/go/apps/api/openapi/openapi.json`
4. Test each endpoint

## Important Notes

- All new endpoints will use the `POST` method only
- All new routes will have a `/v2` prefix
- All responses must follow the "meta" + "data" structure pattern
- Rename "ownerId" fields to "externalId" for consistency
- Implement stricter validation and better field descriptions
- Error responses should follow the new API's error format
- Focus only on routes marked as "Pending" in the migration plan

## Routes to Migrate

### Keys

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/keys.verifyKey` | `/v2/keys.verifyKey` | Completed | Core verification functionality; no root key auth needed but apiId required |
| `/v1/keys.createKey` | `/v2/keys.createKey` | Completed | |
| `/v1/keys.getKey` | `/v2/keys.getKey` | Pending | |
| `/v1/keys.deleteKey` | `/v2/keys.deleteKey` | Pending | |
| `/v1/keys.updateKey` | `/v2/keys.updateKey` | Pending | |
| `/v1/keys.whoami` | `/v2/keys.whoami` | Pending | |
| `/v1/keys.getVerifications` | `/v2/keys.getVerifications` | Skip | |
| `/v1/keys.updateRemaining` | `/v2/keys.updateRemaining` | Pending | |
| `/v1/keys.addPermissions` | `/v2/keys.addPermissions` | Pending | |
| `/v1/keys.removePermissions` | `/v2/keys.removePermissions` | Pending | |
| `/v1/keys.setPermissions` | `/v2/keys.setPermissions` | Pending | |
| `/v1/keys.addRoles` | `/v2/keys.addRoles` | Pending | |
| `/v1/keys.removeRoles` | `/v2/keys.removeRoles` | Pending | |
| `/v1/keys.setRoles` | `/v2/keys.setRoles` | Pending | |

### APIs

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/apis.createApi` | `/v2/apis.createApi` | Completed | |
| `/v1/apis.getApi` | `/v2/apis.getApi` | Completed | |
| `/v1/apis.deleteApi` | `/v2/apis.deleteApi` | Completed | |
| `/v1/apis.listKeys` | `/v2/apis.listKeys` | Completed | |
| `/v1/apis.deleteKeys` | `/v2/apis.deleteKeys` | Pending | |

### Ratelimits

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/ratelimits.limit` | `/v2/ratelimit.limit` | Completed | |
| `/v1/ratelimits.setOverride` | `/v2/ratelimit.setOverride` | Completed | |
| `/v1/ratelimits.getOverride` | `/v2/ratelimit.getOverride` | Completed | |
| `/v1/ratelimits.listOverrides` | `/v2/ratelimit.listOverrides` | Completed | |
| `/v1/ratelimits.deleteOverride` | `/v2/ratelimit.deleteOverride` | Completed | |

### Permissions

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/permissions.createPermission` | `/v2/permissions.createPermission` | Completed | |
| `/v1/permissions.getPermission` | `/v2/permissions.getPermission` | Completed | |
| `/v1/permissions.listPermissions` | `/v2/permissions.listPermissions` | Completed | |
| `/v1/permissions.deletePermission` | `/v2/permissions.deletePermission` | Completed | |
| `/v1/permissions.createRole` | `/v2/permissions.createRole` | Completed | |
| `/v1/permissions.getRole` | `/v2/permissions.getRole` | Completed | |
| `/v1/permissions.listRoles` | `/v2/permissions.listRoles` | Completed | |
| `/v1/permissions.deleteRole` | `/v2/permissions.deleteRole` | Completed | |

### Identities

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/identities.createIdentity` | `/v2/identities.createIdentity` | Completed | |
| `/v1/identities.getIdentity` | `/v2/identities.getIdentity` | Pending | |
| `/v1/identities.listIdentities` | `/v2/identities.listIdentities` | Pending | |
| `/v1/identities.updateIdentity` | `/v2/identities.updateIdentity` | Pending | |
| `/v1/identities.deleteIdentity` | `/v2/identities.deleteIdentity` | Completed | |

### Analytics

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/analytics.getVerifications` | `/v2/analytics.getVerifications` | Pending | |

### System

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/liveness` | `/v2/liveness` | Completed | GET method |

### Legacy

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| Legacy API routes | - | Skip | These should not be migrated |

## Implementation Steps

1. For each pending route:
   - Extract request/response schemas from the TypeScript implementation
   - Convert schemas to the OpenAPI format used in the Go API
   - Add the path to the OpenAPI spec
   - Add appropriate security, descriptions, and examples

2. For each completed route:
   - Verify the implementation against the original TypeScript version
   - Ensure all necessary fields and functionality are included

3. After completing all routes:
   - Validate the full OpenAPI spec
   - Generate client libraries if needed

## Next Steps

1. Start with the key management endpoints:
   - Implement `/v2/keys.createKey` endpoint first as it's foundational
   - Then implement `/v2/keys.verifyKey` as this is the most critical endpoint

2. Follow with remaining Key operations in priority order:
   - Key retrieval (getKey)
   - Key deletion (deleteKey)
   - Key update (updateKey)
   - Key permissions/roles management

3. Then proceed with remaining Identity endpoints

4. Finally implement Analytics endpoints

## Testing Requirements

Each route implementation must include comprehensive test files that match the existing test patterns in the Go API:

1. Test file structure:
   - For each endpoint, create separate test files for different response codes (e.g., `200_test.go`, `400_test.go`, etc.)
   - Use the Go testharness setup for consistent test environment

2. Test scenarios for each route:
   - Happy path (200 responses) with various valid input combinations
   - Validation errors (400 responses)
   - Authentication failures (401 responses)
   - Authorization failures (403 responses)
   - Not found scenarios (404 responses)
   - Other error cases specific to the endpoint

3. Test coverage:
   - Adapt TypeScript tests to follow Go patterns while preserving all scenarios
   - Ensure equivalent coverage to existing TypeScript tests
   - Add additional tests for Go-specific functionality
   - Cover all edge cases from the original implementation

4. Test harness usage:
   - Utilize the `testutil.NewHarness()` for consistent test setup
   - Use the `CallRoute` helper for making requests
   - Verify database state after operations
   - Check audit logs where applicable
   - Follow Go code style and conventions for assertions

## Schema Conversion Notes

- Ensure consistent error formats
- For complex objects, maintain the same structure where possible
- Update field naming if there are changes in conventions:
  - Change "ownerId" to "externalId" consistently
  - Follow Go API naming conventions
- Implement stricter validation for fields (e.g., prefix length, rate limits)
- Add detailed field descriptions and examples
- Use the standardized "meta" + "data" response format for all endpoints

## OpenAPI Structure Patterns

1. Request Body Pattern:
   ```json
   "V2<Resource><Action>RequestBody": {
     "type": "object",
     "required": ["requiredField1", "requiredField2"],
     "properties": {
       "field1": {
         "type": "string",
         "description": "Description",
         "example": "Example value"
       }
     },
     "additionalProperties": false
   }
   ```

2. Response Data Pattern:
   ```json
   "<Resource><Action>ResponseData": {
     "type": "object",
     "properties": {
       "field1": {
         "description": "Description",
         "type": "string"
       }
     },
     "required": ["field1"]
   }
   ```

3. Response Body Pattern:
   ```json
   "V2<Resource><Action>ResponseBody": {
     "type": "object",
     "required": ["meta", "data"],
     "properties": {
       "meta": {
         "$ref": "#/components/schemas/Meta"
       },
       "data": {
         "$ref": "#/components/schemas/<Resource><Action>ResponseData"
       }
     }
   }
   ```

4. Path Pattern:
   ```json
   "/v2/<resource>.<action>": {
     "post": {
       "tags": ["resource"],
       "operationId": "action",
       "x-speakeasy-name-override": "action",
       "security": [{"rootKey": []}],
       "requestBody": {...},
       "responses": {...}
     }
   }
   ```

## Migration Progress Tracking

- [x] Implement /v2/keys.verifyKey endpoint
- [x] Implement /v2/keys.createKey endpoint
- [ ] Complete remaining pending routes
- [ ] Write comprehensive tests for each route
- [ ] Validate all migrated endpoints
- [ ] Update documentation
- [ ] Test with real clients

## Implementation Learnings

### /v2/keys.verifyKey (Completed)

1. Schema Improvements:
   - Made `apiId` a required field with validation (length 3-255 chars)
   - Simplified the permission query structure to limit it to two layers without recursion
   - Renamed "remaining" to "credits" for clarity
   - Removed deprecated "ratelimit" field, replaced with "ratelimits" array
   - Changed "ownerId" to "externalId" for consistency

2. Response Structure:
   - Implemented the "meta" + "data" pattern for the response
   - Made "valid" and "code" required in the response
   - Added array format for ratelimits to support multiple checks

3. OpenAPI Structure:
   - Used proper JSON Schema validation for input parameters
   - Maintained backward compatibility while improving the structure
   - Added detailed descriptions for all fields
   - Provided extensive examples for each field to improve SDK documentation
   - Included security best practices directly in field descriptions
   - Added detailed enum descriptions for verification result codes

4. Documentation Improvements:
   - Added clear explanations of the relationship between keys and APIs
   - Included implementation notes about performance trade-offs (ratelimits vs credits)
   - Added guidance about never storing keys
   - Recommended using key-value pairs for analytics tags
   - Added detailed descriptions of error codes and their meanings

## Key Routes Migration Plan

### Keys.createKey (Completed)

1. Schema Improvements:
   - Removed the deprecated `ownerId` field
   - Renamed "remaining" to "credits" and organized all credits-related options in their own object
   - Replaced the singular "ratelimit" field with "ratelimits" plural array
   - Removed the "environment" field
   - Made credits.remaining required when credits is specified
   - Ensured all ratelimit objects require name, limit, and duration
   - Added detailed field descriptions and examples
   
2. Response Structure:
   - Implemented the "meta" + "data" pattern for the response
   - Enhanced descriptions for keyId and key fields with security guidance
   - Added detailed examples that demonstrate proper key format

3. Documentation Improvements:
   - Added guidance about key strength through byteLength parameter
   - Included clear security recommendations (e.g., not storing keys)
   - Explained the tradeoffs between credits and ratelimits
   - Added examples for complex objects like meta and ratelimits
   - Provided explanations about when to use features like recoverable keys
   
### Keys.getKey (Next in queue)

1. Define schema components:
   - `V2KeysGetKeyRequestBody` - containing keyId parameter
   - `KeysGetKeyResponseData` - containing key properties
   - `V2KeysGetKeyResponseBody` - wrapping meta and data fields

2. Add path entry:
   - Path: `/v2/keys.getKey`
   - Method: POST
   - Security: rootKey
   - Include standard responses (200, 400, 401, 403, 404, 500)

3. Test cases to implement:
   - Get existing key with different field combinations
   - Test retrieval with various permission scenarios
   - Test not found errors
   - Test authentication and authorization errors

### Keys.verifyKey

1. Define schema components:
   - `V2KeysVerifyKeyRequestBody` - similar to v1 with apiId (required), key, etc.
   - `KeysVerifyKeyResponseData` - containing verification result
   - `V2KeysVerifyKeyResponseBody` - wrapping meta and data fields

2. Add path entry:
   - Path: `/v2/keys.verifyKey`
   - Method: POST
   - Security: optional (verification works without auth)
   - Include standard responses plus specific verification error responses
   - Make apiId a required field (different from v1)

3. Test cases to implement:
   - Verify valid key with various configurations
   - Test ratelimit functionality
   - Test ratelimit override
   - Test permission verification
   - Test role verification
   - Test disabled keys
   - Test expired keys
   - Test keys with remaining uses
   - Test keys with identities
   - Test keys with metadata
   - Test deleted keys/APIs scenarios
   - Verify that requests without apiId are rejected
