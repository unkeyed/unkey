# Unkey v2 API Documentation Overhaul Task

## Overview

We are comprehensively updating the OpenAPI documentation for all v2 API endpoints to make them more useful for developers. The documentation will drive both our documentation pages and generated SDKs.

## Documentation Philosophy

**Serve both beginners and experts.** Documentation should provide clear, accessible entry points for newcomers AND comprehensive details for experts who need to understand the full picture, including architectural decisions and edge cases.

**Clarity is better than terse.** We prefer comprehensive documentation that fully explains what the code does in detail, why it exists and its role in the system, how it relates to other components, what callers need to know about behavior and performance characteristics, when to use it versus alternatives, and what can go wrong and why.

**Every piece of documentation should add substantial value.** If the documentation doesn't teach something beyond what's obvious from the function signature, it needs to be expanded.

**Prioritize practical examples over theory.** Every non-trivial function should include working code examples that developers can copy and adapt. Examples should demonstrate real usage patterns, not artificial toy cases.

**Make functionality discoverable.** Use extensive cross-references to help developers find related functions and understand how pieces fit together. If a function works with or is an alternative to another function, mention that explicitly.

**Write in full sentences, not bullet points.** Code documentation should read like well-written prose that flows naturally. Avoid bullet points for general explanations, behavior descriptions, or conceptual information. Only use bullet points when they genuinely improve readability for specific lists such as error codes, configuration options, or step-by-step procedures. Most documentation should be written as coherent paragraphs that explain concepts thoroughly.

## Communication Style Rules (Based on Vercel API Documentation Analysis)

**Developer-first, pragmatic approach.** Write for confident developers who understand technical concepts. Assume competence without over-explaining basics, but provide context where needed.

**Use action-oriented descriptions.** Start endpoint descriptions with clear verbs like "Create", "Retrieve", "Update", "Delete". Lead with what developers can accomplish, not just what the endpoint does.

**Maintain professional but approachable tone.** Be direct and instructional without unnecessary flourishes. Use confident but not prescriptive language. Avoid condescending phrases like "This is typically used..." or "You'll call this when...". Instead use "Use this endpoint to..." or "Call this when...".

**Structure information consistently.** Follow predictable hierarchy across all endpoints. Present essential information first, with additional details available when needed. Clearly distinguish between required and optional parameters.

**Provide practical, context-aware examples.** Show real-world usage scenarios that are relevant to the specific endpoint. Include clear authentication patterns and multiple request/response examples.

**Use technical precision without intimidation.** Employ exact technical terminology without oversimplification. Proactively explain potential failure scenarios and error conditions. Provide sufficient implementation guidance for successful integration.

**Balance conciseness with completeness.** Focus on essential information upfront with layered detail available. Avoid redundancy across sections while ensuring each endpoint is thoroughly documented within its scope.

## Process

For each endpoint, we:

1. **Ask relevant questions** (one at a time) to understand business context and use cases
2. **Read the handler function** from `@go/apps/api/routes/v2_*/handler.go` to understand implementation
3. **Read test cases** from `@go/apps/api/routes/v2_*/*_test.go` to understand behavior and edge cases
4. **Update OpenAPI specs** in `@go/apps/api/openapi/spec/paths/v2/` with comprehensive documentation

## What We Document for Each Endpoint

- **Clear explanation** of what the endpoint does and why it exists
- **Required parameters** and their validation rules
- **Business context** and real-world use cases
- **Required permissions** (found by studying handler code, documented as simple list in description)
- **Non-obvious side effects** (infrastructure provisioning, audit logs, etc.)
- **Practical examples** with realistic data and detailed descriptions
- **Comprehensive response documentation** explaining what developers need to know

## File Structure

### OpenAPI Specs Location
```
@go/apps/api/openapi/spec/paths/v2/
├── apis/
│   ├── createApi/
│   │   ├── index.yaml                           # Endpoint definition
│   │   ├── V2ApisCreateApiRequestBody.yaml     # Request schema
│   │   ├── V2ApisCreateApiResponseBody.yaml    # Response schema
│   │   └── V2ApisCreateApiResponseData.yaml    # Response data schema
│   ├── deleteApi/
│   ├── getApi/
│   └── listKeys/
├── keys/
│   ├── createKey/
│   ├── verifyKey/
│   ├── getKey/
│   ├── updateKey/
│   ├── deleteKey/
│   ├── addPermissions/
│   ├── removePermissions/
│   ├── setPermissions/
│   ├── addRoles/
│   ├── removeRoles/
│   ├── setRoles/
│   └── updateCredits/
├── identities/
├── permissions/
├── ratelimit/
└── liveness/
```

### Handler Code Location
```
@go/apps/api/routes/
├── v2_apis_create_api/
│   ├── handler.go           # Implementation
│   ├── 200_test.go         # Success cases
│   ├── 400_test.go         # Bad request cases
│   ├── 401_test.go         # Unauthorized cases
│   ├── 403_test.go         # Forbidden cases
│   └── 404_test.go         # Not found cases
├── v2_apis_delete_api/
├── v2_keys_create_key/
└── ... (similar structure for all endpoints)
```

## All v2 Endpoints (36 total)

### APIs Management (4 endpoints) - ✅ **GROUP COMPLETED**
- `POST /v2/apis` (createApi) - ✅ **COMPLETED** (Flexible API referencing, proper multiline strings, examples in schema files)
- `POST /v2/apis/get` (getApi) - ✅ **COMPLETED** (Basic API info retrieval, simple functionality, minimal use cases)
- `POST /v2/apis/delete` (deleteApi) - ✅ **COMPLETED** (Cleanup use cases, delete protection, immediate key invalidation)
- `POST /v2/apis/listKeys` (listKeys) - ✅ **COMPLETED** (Dashboard listing patterns, identity filtering, pagination, decrypt functionality)

### Key Management (6 endpoints) - ✅ **GROUP COMPLETED**
- `POST /v2/keys` (createKey) - ✅ **COMPLETED** (Tier-based examples, proper file organization, multiline strings)
- `POST /v2/keys/verify` (verifyKey) - ✅ **COMPLETED** (Comprehensive documentation, examples in schema files, fixed response codes)
- `POST /v2/keys/get` (getKey) - ✅ **COMPLETED** (Dashboard/playground use cases, decrypt functionality, proper examples)
- `POST /v2/keys/update` (updateKey) - ✅ **COMPLETED** (Plan change scenarios, partial updates, comprehensive field coverage)
- `POST /v2/keys/delete` (deleteKey) - ✅ **COMPLETED** (User deletion requests, account deletion workflows, soft vs permanent deletion)
- `POST /v2/keys/updateCredits` (updateCredits) - ✅ **COMPLETED** (Quota management, plan changes, credit operations)

### Permission Management (14 endpoints) - ✅ **GROUP COMPLETED**
#### Permissions (4 endpoints) - ✅ **SUBGROUP COMPLETED**
- `POST /v2/permissions` (createPermission) - ✅ **COMPLETED** (New resource/action permissions, hierarchical naming, RBAC foundation)
- `POST /v2/permissions/get` (getPermission) - ✅ **COMPLETED** (Simple permission details retrieval, minimal functionality)
- `POST /v2/permissions/list` (listPermissions) - ✅ **COMPLETED** (List permissions for dashboard interfaces, minimal functionality)
- `POST /v2/permissions/delete` (deletePermission) - ✅ **COMPLETED** (Remove permissions and cleanup assignments, minimal functionality)

#### Roles (4 endpoints) - ✅ **SUBGROUP COMPLETED**
- `POST /v2/permissions/roles` (createRole) - ✅ **COMPLETED** (Group permissions into reusable roles, standardized access patterns)
- `POST /v2/permissions/roles/get` (getRole) - ✅ **COMPLETED** (Simple role details retrieval, minimal functionality)
- `POST /v2/permissions/roles/list` (listRoles) - ✅ **COMPLETED** (List roles for dashboard interfaces, minimal functionality)
- `POST /v2/permissions/roles/delete` (deleteRole) - ✅ **COMPLETED** (Remove roles and cleanup assignments, minimal functionality)

#### Key Permission Operations (6 endpoints) - ✅ **SUBGROUP COMPLETED**
- `POST /v2/keys/addPermissions` (addPermissions) - ✅ **COMPLETED** (Add permissions to keys, role-based access expansion, proper permission requirements)
- `POST /v2/keys/removePermissions` (removePermissions) - ✅ **COMPLETED** (Remove permissions from keys, privilege downgrading, remaining permissions response)
- `POST /v2/keys/setPermissions` (setPermissions) - ✅ **COMPLETED** (Replace all permissions atomically, complete permission state management)
- `POST /v2/keys/addRoles` (addRoles) - ✅ **COMPLETED** (Add roles to keys, role-based privilege promotion, comprehensive examples)
- `POST /v2/keys/removeRoles` (removeRoles) - ✅ **COMPLETED** (Remove roles from keys, role-based access revocation, remaining roles response)
- `POST /v2/keys/setRoles` (setRoles) - ✅ **COMPLETED** (Replace all roles atomically, complete role state management)

### Identity Management (5 endpoints) - ✅ **GROUP COMPLETED**
- `POST /v2/identities` (createIdentity) - ✅ **COMPLETED** (Resource sharing across keys, metadata management, rate limit association)
- `POST /v2/identities/get` (getIdentity) - ✅ **COMPLETED** (Retrieve identity details, metadata access, dashboard building)
- `POST /v2/identities/list` (listIdentities) - ✅ **COMPLETED** (Browse all identities, management interfaces, pagination support)
- `POST /v2/identities/update` (updateIdentity) - ✅ **COMPLETED** (Update metadata and rate limits, subscription changes, partial updates)
- `POST /v2/identities/delete` (deleteIdentity) - ✅ **COMPLETED** (Permanent removal, compliance support, key preservation)

### Rate Limiting (5 endpoints) - ✅ **GROUP COMPLETED**
- `POST /v2/ratelimit/limit` (limit) - ✅ **COMPLETED** (Core rate limiting check, namespace-based limiting, override support, sliding window implementation)
- `POST /v2/ratelimit/setOverride` (setOverride) - ✅ **COMPLETED** (Create/update custom rate limits, wildcard patterns, tiered limiting policies)
- `POST /v2/ratelimit/getOverride` (getOverride) - ✅ **COMPLETED** (Retrieve override details, audit configurations, troubleshooting support)
- `POST /v2/ratelimit/listOverrides` (listOverrides) - ✅ **COMPLETED** (List all namespace overrides, pagination support, policy management)
- `POST /v2/ratelimit/deleteOverride` (deleteOverride) - ✅ **COMPLETED** (Remove overrides, revert to defaults, cleanup outdated rules)

### System Health (1 endpoint) - ✅ **GROUP COMPLETED**
- `GET /v2/liveness` (liveness) - ✅ **COMPLETED** (Service health check, monitoring support, no authentication required)

## Key Context Gathered So Far

### API Creation (createApi)
- **Primary use cases**: Environment separation (dev/staging/prod) and product separation
- **Most common pattern**: Users create separate APIs for local dev, staging, and production
- **API names**: Not required to be unique within workspace (multiple APIs can have same name)
- **Required permission**: `api.*.create_api`
- **Side effects**: 
  - Creates keyring infrastructure automatically
  - Provisions database entries
  - Sets up audit logging infrastructure
  - All resources immediately available for use
- **Response**: Returns unique API ID that must be stored securely for all future operations

### Key Retrieval (getKey)
- **Primary use cases**: Dashboard building (showing key details) and API playground functionality (testing with decrypted keys)
- **Dashboard patterns**: Most commonly used for displaying key metadata, status, permissions, and usage in management interfaces
- **API playground patterns**: Decrypt functionality enables testing API calls directly in dashboards without copy/paste
- **Two identification methods**: Can use either database `keyId` (more common) or actual key string (useful for support)
- **Security considerations**: Decrypt functionality should be used sparingly - storing ciphertext is less secure than hashes
- **Required permissions**: `api.*.read_key` for basic info, plus `api.*.decrypt_key` for plaintext retrieval
- **Response data**: Returns all key metadata including permissions, roles, credits, rate limits, identity info, and optionally plaintext key

### Key Listing (listKeys)
- **Primary use cases**: Dashboard interfaces showing "all keys for user X" with identity-based filtering
- **Common filtering pattern**: Filter by `externalId` on API side, then optional client-side filtering for smaller result sets
- **Pagination support**: Configurable limits with cursor-based pagination for APIs with larger numbers of keys
- **Decrypt functionality**: Same as getKey - can retrieve plaintext values for recoverable keys with proper permissions
- **Required permissions**: Both `api.*.read_key` and `api.*.read_api` (or specific API equivalents), plus `api.*.decrypt_key` for decryption
- **Response structure**: Array of key objects with same detailed metadata as getKey, plus pagination metadata

### API Deletion (deleteApi)
- **Primary use cases**: Cleanup when finished with an API - shutting down dev/staging environments, retiring services, removing unused resources
- **Immediate effects**: API marked as deleted, all associated keys invalidated and fail verification with `code=NOT_FOUND`
- **Delete protection**: Safety mechanism prevents accidental deletion of critical APIs - must be disabled first (returns 412 if enabled)
- **Soft deletion**: API is marked as deleted rather than physically removed, maintaining referential integrity
- **Required permissions**: `api.*.delete_api` or `api.<api_id>.delete_api`
- **Audit logging**: Creates audit trail for API deletion with actor information and timestamp

### API Information Retrieval (getApi)
- **Primary use cases**: Basic information retrieval - minimal usage, mainly for completeness
- **Simple functionality**: Returns only API ID and name (basic identifying information)
- **Potential scenarios**: Verify API exists, get human-readable name from ID, confirm access to namespace
- **Minimal response**: No complex metadata, just essential identifying details
- **Required permissions**: `api.*.read_api` or `api.<api_id>.read_api`
- **Low usage pattern**: Not heavily used in practice but available when needed

### Key Updates (updateKey)
- **Primary use cases**: Responding to user plan changes, subscription updates, role modifications, account status changes
- **Plan management scenarios**: Upgrade users from free to paid (increase limits/credits), downgrade cancelled subscriptions (reduce limits), adjust permissions for role changes
- **Administrative actions**: Temporarily disable keys for suspended accounts, update metadata for current user status, modify identity associations
- **Partial update support**: Only specify fields you want to change, explicitly set null to clear fields, preserves unchanged properties
- **Permission handling**: Replaces entire permission/role sets rather than adding to existing ones - use dedicated add/remove endpoints for incremental changes
- **Required permissions**: `api.*.update_key` or `api.<api_id>.update_key` for the target API
- **Side effects**: Creates audit log entries, auto-creates missing identities/permissions, immediate effect with 30-second edge propagation delay

### Key Deletion (deleteKey)
- **Primary use cases**: User-requested key deletion, complete account deletion workflows, permanent removal requirements
- **User scenarios**: Users click "Delete" in dashboards, users delete entire accounts, cleanup of test/development keys no longer needed
- **Deletion modes**: Soft delete (default) preserves records for audit trails, permanent delete completely removes all data for compliance
- **Immediate effects**: Keys become invalid instantly, all permissions/roles removed, metadata cleared, rate limit tracking stopped
- **Vs temporary disabling**: Use `updateKey` with `enabled: false` for temporary access control - deletion is for permanent removal
- **Required permissions**: `api.*.delete_key` or `api.<api_id>.delete_key` for the target API
- **Side effects**: Creates audit log entries, removes key from cache, immediate effect with 30-second edge propagation delay

### Credit Updates (updateCredits)
- **Primary use cases**: Quota management in response to plan changes, credit purchases, billing cycle resets, policy enforcement
- **Plan scenarios**: Upgrade users to higher credit tiers, set unlimited usage for enterprise plans, reset monthly quotas at billing cycles
- **Credit operations**: Set absolute credit values, increment credits for purchases, decrement credits for refunds or violations
- **Immediate effects**: Credit changes apply instantly to new verifications, unlimited mode removes all usage restrictions
- **Operation types**: 'set' replaces current credits or enables unlimited usage, 'increment' adds to existing balance, 'decrement' subtracts from balance (minimum zero)
- **Required permissions**: `api.*.update_key` or `api.<api_id>.update_key` for the target API
- **Side effects**: Creates audit log entries, removes key from cache, automatic refill clearing when setting unlimited, immediate effect with 30-second edge propagation delay

### Permission Creation (createPermission)
- **Primary use cases**: Defining new resources or actions in access control system, expanding API with new endpoints, implementing granular user permissions
- **Resource scenarios**: Adding new features requiring access control, creating administrative actions, organizing existing functionality into discrete permissions
- **Naming patterns**: Use hierarchical naming like 'documents.read', 'admin.users.delete', 'billing.invoices.create' for clear organization
- **RBAC foundation**: Permissions are building blocks that can be granted directly to keys or organized into roles for easier management
- **Uniqueness requirement**: Permission names must be unique within workspace to prevent conflicts during assignment
- **Required permissions**: `rbac.*.create_permission` for workspace-level permission creation
- **Side effects**: Creates audit log entries, makes permission immediately available for assignment to keys or roles

### Permission Retrieval (getPermission)
- **Primary use cases**: Simple permission checking and verification, minimal functionality for inspecting permission details
- **Basic functionality**: Returns permission name, slug, description, and creation date for verification purposes
- **Simple verification**: Check if permission exists and review its configuration before assignment or updates
- **Required permissions**: `rbac.*.read_permission` for workspace-level permission reading
- **Minimal response**: Straightforward permission metadata without complex relationships or usage data

### Role Creation (createRole)
- **Primary use cases**: Grouping related permissions for easier management, establishing standardized access patterns, simplifying permission assignments at scale
- **Permission grouping**: Bundle related permissions like 'admin', 'editor', or 'billing_manager' for consistent assignment to multiple keys
- **Access patterns**: Create reusable roles that represent common user types or job functions in your application
- **Scale management**: Reduce complexity of individual permission management by creating role-based permission bundles
- **Uniqueness requirement**: Role names must be unique within workspace to prevent conflicts during assignment
- **Required permissions**: `rbac.*.create_role` for workspace-level role creation
- **Side effects**: Creates audit log entries, makes role immediately available for assignment to keys

## Sample Documentation Update

Here's an example of how we updated the createApi endpoint following our philosophy:

**Before**: Brief, bullet-pointed description
**After**: Comprehensive prose explaining:
- What the endpoint does and why it exists
- Automatic infrastructure provisioning
- Three primary organizational purposes (environment/service/product separation)
- Real-world scenarios with specific examples
- Critical importance of storing the API ID
- Required permissions clearly documented
- Multiple realistic request/response examples with detailed descriptions

## Key Learnings So Far

### Documentation Structure Best Practices
- **Examples belong in schema files, not index.yaml**: Request examples should be in the request body schema file, response examples in the response body schema file. This keeps the main index.yaml focused on endpoint definitions and makes examples more maintainable.
- **Use proper multiline YAML syntax**: Use `description: |` instead of `description: |-` or inline multiline strings for better readability and consistent formatting.
- **Shorter, readable IDs in examples**: Use 8-16 character IDs like `api_1234abcd` and `key_5678efgh` instead of extremely long random strings that are hard to read.
- **Consistent key prefixes**: Use `sk_` prefix for API keys in examples to match real-world conventions.

### Business Context Insights
- **API creation patterns**: Most users create APIs for environment separation (dev/staging/prod) and product separation. Names don't need to be unique within workspace.
- **Key creation tiers**: Free tier users get limited credits and restrictive rate limits to encourage upgrades. Paid users get higher limits. Enterprise gets custom permissions and unlimited credits.
- **Flexible API referencing**: Users can reference APIs by either ID or name in subsequent operations, plus use scoped permissions like `api.<api_id>.verify_key` to restrict operations without specifying API identifiers in each request.
- **Credit-based billing**: Credits are deducted after security checks pass (not immediately), enabling consumption-based pricing where users are only charged for successful verifications.

### Permission Documentation Standards
- **Use "one of the following" pattern**: Make it clear when users need either/or permissions rather than all permissions listed.
- **Include both wildcard and scoped options**: Document both `api.*.action` (all APIs) and `api.<api_id>.action` (specific API) permission patterns.
- **List permissions in descriptions**: Include required permissions directly in the description text rather than custom OpenAPI directives so they appear in rendered documentation.

### Technical Implementation Details
- **verifyKey always returns HTTP 200**: Success/failure determined by `valid` field in response data, not HTTP status code. This prevents information leakage about key existence.
- **Response codes corrected**: Fixed inconsistent response codes like `USAGE_EXCEEDED` → `INSUFFICIENT_CREDITS` and `INSUFFICIENT_PERMISSIONS` → `FORBIDDEN` to match actual handler implementation.
- **Credits timing**: Credits are deducted after all security checks pass, ensuring users are only charged for successful verifications.

### Documentation Philosophy Applied
- **Business context over technical details**: Each endpoint now explains not just how to use it, but when and why developers would use it in real-world scenarios.
- **Practical, tier-based examples**: Examples reflect actual usage patterns like free/paid/enterprise tiers rather than generic scenarios.
- **Clear error handling**: Documented all possible failure scenarios with specific response codes and when they occur.
- **Cross-references**: Connected related concepts like API creation → key creation → key verification workflow.
- **Avoid filler words**: Removed words like "comprehensive" that don't add value - focus on being direct and specific.

## Next Steps

Continue with the remaining 29 endpoints, following the same process:
1. Ask contextual questions about business use cases
2. Read handler and test files to understand implementation
3. Update OpenAPI documentation with comprehensive details
4. Ensure proper file organization (examples in schema files)
5. Use consistent formatting (multiline strings, readable IDs)
6. Move to next endpoint

The goal is to make each endpoint's documentation comprehensive enough that developers can understand not just how to call it, but when to use it, what it does behind the scenes, and how it fits into their overall API key management strategy.

## 🎉 TASK COMPLETED! 

**Final Status: 36 of 36 endpoints completed (100%)**

All v2 API endpoints have been comprehensively documented with:
- Clear business context and use cases
- Proper permission requirements 
- Detailed side effects documentation
- Comprehensive error handling examples
- Practical, real-world examples
- Consistent formatting and structure

### Groups Completed:
- **APIs Management**: 4/4 endpoints ✅ COMPLETED
- **Key Management**: 6/6 endpoints ✅ COMPLETED  
- **Permission Management**: 14/14 endpoints ✅ COMPLETED
- **Rate Limiting**: 5/5 endpoints ✅ COMPLETED
- **Identity Management**: 5/5 endpoints ✅ COMPLETED
- **System Health**: 1/1 endpoint ✅ COMPLETED

The v2 API documentation overhaul is now complete and ready to drive both documentation pages and generated SDKs with comprehensive, developer-friendly content.