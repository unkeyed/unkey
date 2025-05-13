# OpenAPI Migration Plan: TypeScript API to Go API

This document outlines the plan for migrating our OpenAPI specifications from the old TypeScript API to the new Go API with v2 endpoints. The migration is focused on creating a consistent, well-documented API specification that will be used to generate client SDKs and documentation.

## Summary of Progress (Last Updated: April 2023)

**Current Status:** 5 of 12+ planned endpoints completed (42% complete)

**Completed Endpoints:**
1. ✅ `/v2/keys.verifyKey` - Core verification functionality
2. ✅ `/v2/keys.createKey` - Key creation with enhanced options
3. ✅ `/v2/keys.getKey` - Detailed key information retrieval
4. ✅ `/v2/keys.deleteKey` - Key deletion with soft/permanent options
5. ✅ `/v2/keys.updateKey` - Partial key property updates

**Up Next:** `/v2/keys.whoami` endpoint

**Key Improvements Made:**
- Standardized response structures with meta/data pattern
- Enhanced field validation and documentation
- Renamed fields for clarity (ownerId → externalId)
- Reorganized related properties into logical groups
- Provided comprehensive examples and descriptions

## Project Overview

**Objective:** Create OpenAPI specifications for all v2 endpoints in the Go API that match the functionality of the existing v1 TypeScript API while improving documentation, consistency, and usability.

**Timeline:** No fixed deadline, but prioritizing key management endpoints first

**Repository Structure:**
- Old TypeScript API: `unkey/apps/api/src/routes/`
- New Go API: `unkey/go/apps/api/`
- OpenAPI Specification: `unkey/go/apps/api/openapi/openapi.json`

## Migration Approach

1. Identify all existing routes in the TypeScript API
2. Map each route to a corresponding v2 route
3. Update the OpenAPI schema in `unkey/go/apps/api/openapi/openapi.json`
4. Test each endpoint

## Detailed Implementation Process

For each endpoint:

1. **Research the TypeScript implementation**:
   - Read the route handler in `unkey/apps/api/src/routes/v1_*.ts`
   - Understand request/response structure and validation
   - Note any special behaviors or edge cases

2. **Design the OpenAPI schema**:
   - Create request body schema (`V2<Resource><Action>RequestBody`)
   - Create response data schema (`<Resource><Action>ResponseData`)
   - Create response body schema (`V2<Resource><Action>ResponseBody`)
   - Add path definition with appropriate responses

3. **Add to OpenAPI specification**:
   - Open `unkey/go/apps/api/openapi/openapi.json`
   - Add schemas to the `components.schemas` section
   - Add path definition to the `paths` section
   - Ensure proper JSON formatting

4. **Update migration plan**:
   - Mark the endpoint as completed
   - Document any significant changes or decisions
   - Prepare for the next endpoint

5. **Validate the specification**:
   - Check for valid JSON format
   - Verify consistent patterns are followed
   - Test with OpenAPI validation tools if available

## Important Notes

- All new endpoints will use the `POST` method only (exception: liveness check remains GET)
- All new routes will have a `/v2` prefix
- All responses must follow the "meta" + "data" structure pattern
- Rename "ownerId" fields to "externalId" for consistency
- Implement stricter validation and better field descriptions
- Error responses should follow the new API's error format
- Focus only on routes marked as "Pending" in the migration plan

## Development Environment Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/unkey/unkey.git
   ```

2. **Navigate to the API project**:
   ```bash
   cd unkey/go/apps/api
   ```

3. **Review existing OpenAPI specification**:
   ```bash
   cat openapi/openapi.json
   ```

4. **Validate OpenAPI changes**:
   - Use an OpenAPI validator tool or VS Code extension
   - Or paste the JSON into the Swagger Editor (https://editor.swagger.io/)

5. **Testing process**:
   - Update the migration plan after each endpoint is implemented
   - Manual validation recommended before pushing changes
   - Coordinate with the team for any significant structural changes

## Routes to Migrate

### Keys

| TypeScript Route | Go Route | Status | Notes |
|------------------|----------|--------|-------|
| `/v1/keys.verifyKey` | `/v2/keys.verifyKey` | Completed | Core verification functionality; no root key auth needed but apiId required |
| `/v1/keys.createKey` | `/v2/keys.createKey` | Completed | |
| `/v1/keys.getKey` | `/v2/keys.getKey` | Completed | |
| `/v1/keys.deleteKey` | `/v2/keys.deleteKey` | Completed | |
| `/v1/keys.updateKey` | `/v2/keys.updateKey` | Completed | |
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

1. Continue with key management endpoints in this order:
   - Implement `/v2/keys.whoami` endpoint next
   - Proceed with key permissions endpoints (addPermissions, removePermissions, setPermissions)
   - Implement key roles endpoints (addRoles, removeRoles, setRoles)
   - Complete remaining key operations (updateRemaining)

2. Follow with Identity endpoints:
   - Focus on the getIdentity endpoint first
   - Then implement listIdentities
   - Finally implement updateIdentity

3. Complete the Analytics endpoints:
   - Implement getVerifications endpoint

4. Conduct comprehensive validation and testing across all implemented endpoints

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
- Proactively enhance descriptions and examples without being prompted:
  - Include real-world usage scenarios
  - Add security warnings where appropriate
  - Explain performance implications and tradeoffs
  - Clarify relationships between different components
  - Use rich, nested examples that demonstrate practical applications

## Field Changes and Standardization

The following changes should be applied consistently across all endpoints:

| Old Field (v1) | New Field (v2) | Notes |
|----------------|---------------|-------|
| `ownerId` | `externalId` | Consistently renamed for clarity |
| `remaining` | `credits.remaining` | Moved into credits object |
| `refill` | `credits.refill` | Moved into credits object |
| `ratelimit` | `ratelimits[]` | Changed to array of named ratelimits |
| `environment` | Removed | Field has been removed in v2 |

### Common Required Fields

- Request bodies: Usually require the resource identifier (e.g., `keyId`, `apiId`)
- Response bodies: Always require `meta` with `requestId`
- Data objects: Required fields vary by endpoint but are explicitly specified

### Standard Response Structure

All endpoints follow this response pattern:
```json
{
  "meta": {
    "requestId": "req_123abc"
  },
  "data": {
    // Endpoint-specific data
  }
}
```

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

## Common Pitfalls and Solutions

1. **JSON Schema Validation Issues**:
   - Always run validation after changes
   - Watch for duplicate keys or missing commas
   - Make sure required fields actually exist in properties

2. **Inconsistent Naming**:
   - Follow established patterns for schema names
   - Maintain consistent casing (camelCase for properties)
   - Use descriptive names for schemas and properties

3. **Documentation Errors**:
   - Ensure examples match the described schema
   - Verify enum values are actually valid options
   - Make sure all required fields are documented

4. **Common JSON Errors**:
   - Trailing commas (not allowed in JSON)
   - Missing commas between properties
   - Unmatched brackets or braces
   - Quotes around property names (required)

5. **Schema Hierarchy**:
   - Maintain proper $ref paths
   - Avoid circular references
   - Place new schemas in the appropriate section

## Migration Progress Tracking

### Completed Endpoints:
- [x] `/v2/keys.verifyKey` - For verifying API keys
- [x] `/v2/keys.createKey` - For creating new API keys
- [x] `/v2/keys.getKey` - For retrieving API key details
- [x] `/v2/keys.deleteKey` - For deleting API keys
- [x] `/v2/keys.updateKey` - For updating API key properties

### Next Endpoints to Implement:
1. [ ] `/v2/keys.whoami` - For identifying the current key (NEXT)
2. [ ] `/v2/keys.addPermissions` - For adding permissions to a key
3. [ ] `/v2/keys.removePermissions` - For removing permissions from a key
4. [ ] `/v2/keys.setPermissions` - For setting all permissions on a key
5. [ ] `/v2/keys.addRoles` - For adding roles to a key
6. [ ] `/v2/keys.removeRoles` - For removing roles from a key
7. [ ] `/v2/keys.setRoles` - For setting all roles on a key
8. [ ] `/v2/keys.updateRemaining` - For updating key usage credits

### Remaining Tasks:
- [ ] Complete all pending routes
- [ ] Write comprehensive tests for each route
- [ ] Validate all migrated endpoints
- [ ] Update documentation
- [ ] Test with real clients
- [ ] Get final approval from the team

## Implementation Priority

1. Key management endpoints (currently in progress)
2. API management endpoints
3. Identity management endpoints
4. Analytics endpoints

Each endpoint should be implemented sequentially, updating this document after completion.

## Implementation Details and Learnings

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

5. Source Files:
   - TypeScript original: `unkey/apps/api/src/routes/v1_keys_verifyKey.ts`
   - OpenAPI changes in: `unkey/go/apps/api/openapi/openapi.json`

6. Key Considerations:
   - This endpoint does NOT require authentication (unlike most other endpoints)
   - The endpoint will return a 200 OK even when verification fails (check the `valid` field)
   - The verification codes follow a specific enum of possible values

## Key Routes Migration Plan

### /v2/keys.createKey (Completed)

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

4. Source Files:
   - TypeScript original: `unkey/apps/api/src/routes/v1_keys_createKey.ts`
   - OpenAPI changes in: `unkey/go/apps/api/openapi/openapi.json`

5. Key Considerations:
   - Authentication with a root key is required with permissions to create keys
   - The `apiId` field is mandatory and validates against accessible APIs
   - The response includes both the key ID (for reference) and the actual key (for distribution)
   - The encryption and security model is complex and detailed in the docs

### /v2/keys.getKey (Completed)

1. Schema Improvements:
   - Converted the GET endpoint with query parameters to a POST endpoint with a JSON body
   - Changed "ownerId" to "externalId" for consistency
   - Restructured "remaining" and "refill" into a credits object
   - Converted "ratelimit" to an array of "ratelimits" with named limits
   - Made required fields explicit in both request and response schemas
   - Added comprehensive field validations

2. Response Structure:
   - Implemented the standard "meta" + "data" pattern
   - Enhanced the key object structure with clear field groupings
   - Included opt-in plaintext key retrieval with proper security warnings
   - Maintained all fields from the v1 response in a more organized structure

3. Documentation Improvements:
   - Added detailed descriptions explaining the purpose of each field
   - Included rich examples showing realistic data structures
   - Added security warnings about handling decrypted keys
   - Clarified the relationship between keys and identities
   - Explained how different ratelimit and credit configurations work

4. Source Files:
   - TypeScript original: `unkey/apps/api/src/routes/v1_keys_getKey.ts`
   - Referenced schema: `unkey/apps/api/src/routes/schema.ts`
   - OpenAPI changes in: `unkey/go/apps/api/openapi/openapi.json`

5. Key Considerations:
   - Authentication with appropriate permissions is required
   - Decryption of keys requires special permissions and is opt-in only
   - The response contains potentially sensitive information and should be handled accordingly
   - Workspace isolation is enforced for security
   
### Keys.deleteKey (Next in queue)

1. Schema components:
   - Created `V2KeysGetKeyRequestBody` with keyId and decrypt parameters
   - Implemented `KeysGetKeyResponseData` with comprehensive key properties
   - Added `V2KeysGetKeyResponseBody` with the meta/data wrapper pattern

2. Path entry:
   - Added path: `/v2/keys.getKey`
   - Used POST method with JSON body
   - Required rootKey authentication
   - Included all standard responses with proper status codes and descriptions

3. Test scenarios to consider:
   - Retrieving keys with various field combinations (with/without meta, permissions, etc.)
   - Testing permission boundary enforcement (workspace isolation)
   - Handling not found cases gracefully
   - Testing key decryption with proper permissions
   - Verifying authentication and authorization requirements

### /v2/keys.deleteKey (Completed)

1. Schema Improvements:
   - Created a simple but effective request schema with keyId and permanent parameters
   - Added clear descriptions about soft vs permanent deletion
   - Maintained an empty response body structure with proper meta/data pattern
   - Used consistent field naming and validation rules
   
2. Documentation Improvements:
   - Added detailed explanation of deletion propagation time
   - Clarified the difference between soft and permanent deletion
   - Explained use cases for permanent deletion (hash conflicts, complete removal)
   - Added clear examples with proper formatting
   
3. Design Considerations:
   - Kept the API simple with minimal required parameters
   - Maintained backward compatibility with v1 behavior
   - Added appropriate error responses for all potential error cases
   - Included cache invalidation notes in the documentation

4. Source Files:
   - TypeScript original: `unkey/apps/api/src/routes/v1_keys_deleteKey.ts`
   - OpenAPI changes in: `unkey/go/apps/api/openapi/openapi.json`

5. Key Considerations:
   - Authentication with delete permissions is required
   - Soft deletion is the default, permanent deletion is opt-in
   - There's a propagation delay of up to 30 seconds for deletion to take effect in all regions
   - Audit logs are created for deletion operations
   - Cache invalidation happens automatically

### /v2/keys.updateKey (Completed)

1. Schema Improvements:
   - Created a comprehensive request schema with all updatable properties
   - Used nullable types to allow explicitly removing features
   - Removed roles and permissions fields (handled by specialized endpoints)
   - Replaced deprecated `ratelimit` field with array-based `ratelimits`
   - Maintained the empty response structure with proper meta/data pattern
   
2. Documentation Improvements:
   - Clearly documented the partial update behavior (omit vs. null)
   - Added detailed field descriptions with examples
   - Included propagation time estimates for changes
   - Provided clear examples of valid update operations
   
3. Design Considerations:
   - Simplified the endpoint by moving permission/role management to specialized endpoints
   - Used JSON types to enforce field validations
   - Maintained consistency with other endpoints' field structures
   - Kept backward compatibility where appropriate while removing deprecated fields

4. Source Files:
   - TypeScript original: `unkey/apps/api/src/routes/v1_keys_updateKey.ts`
   - OpenAPI changes in: `unkey/go/apps/api/openapi/openapi.json`

5. Key Considerations:
   - Used nullable types to distinguish between "remove this feature" and "don't change"
   - Authentication with update permissions is required
   - Simplified interface compared to v1 by removing role/permission management
   - Improved documentation of validation rules
   - Cache invalidation period is clearly documented

### /v2/keys.whoami (NEXT IN QUEUE)

#### Implementation Plan

1. Schema components:
   - `V2KeysUpdateKeyRequestBody` - with keyId and updateable properties
   - `KeysUpdateKeyResponseData` - with updated key information
   - `V2KeysUpdateKeyResponseBody` - wrapping meta and data fields

2. Path entry:
   - Path: `/v2/keys.updateKey`
   - Method: POST
   - Security: rootKey authentication
   - Include standard responses (200, 400, 401, 403, 404, 500)

3. Design considerations:
   - Which fields should be updatable and which should not
   - Partial updates vs. full replacements
   - Validation rules for each field
   - Proper documentation of update behaviors

4. Source files to reference:
   - TypeScript original: `unkey/apps/api/src/routes/v1_keys_updateKey.ts`

## Additional Resources

### Reference Implementations

- **TypeScript API Routes**: `unkey/apps/api/src/routes/`
- **Go API Routes**: `unkey/go/apps/api/routes/`
- **OpenAPI Specification**: `unkey/go/apps/api/openapi/openapi.json`

### Documentation

- **Field Naming Conventions**: See the "Field Changes and Standardization" section
- **Response Patterns**: See the "Standard Response Structure" section
- **Schema Patterns**: See the "OpenAPI Structure Patterns" section

### Validation

- Recommend using [Swagger Editor](https://editor.swagger.io/) for validation
- VS Code extensions like "OpenAPI (Swagger) Editor" are helpful
- Validate changes before committing

### Handover Notes for Next Developer

1. Start by reviewing the five completed endpoints in the migration plan
2. The next endpoint to implement is `/v2/keys.whoami` - begin by examining the TypeScript implementation in `unkey/apps/api/src/routes/v1_keys_whoami.ts`
3. Follow these implementation steps:
   - Create the request/response schemas in the OpenAPI spec
   - Add the path definition
   - Ensure thorough field descriptions and examples
   - Update the migration plan after completion
4. Key patterns to maintain:
   - All responses use meta/data pattern
   - All descriptions should be comprehensive with examples
   - Remove deprecated fields (like "ownerId")
   - Organize related properties into logical groups
   - Follow existing naming and structure conventions
5. Refer to the "Implementation Details and Learnings" sections of completed endpoints for guidance
6. Consider separating complex functionality into dedicated endpoints (as we did with permissions/roles)
