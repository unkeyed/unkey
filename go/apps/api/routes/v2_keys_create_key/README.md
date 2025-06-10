# v2/keys.createKey Endpoint Implementation

## Overview

This endpoint implements the `POST /v2/keys.createKey` API for creating new API keys in the Unkey system. It follows the OpenAPI specification and provides comprehensive key creation functionality with security, validation, and audit logging.

## Features Implemented

### Core Functionality
- ✅ **Key Generation**: Cryptographically secure API key generation using configurable byte lengths
- ✅ **Prefix Support**: Optional custom prefixes for key identification
- ✅ **API Isolation**: Keys are isolated by API to prevent cross-environment usage
- ✅ **Workspace Isolation**: Strict workspace boundaries enforced
- ✅ **Audit Logging**: Complete audit trail for key creation and permission/role assignments

### Security Features
- ✅ **Root Key Authentication**: Verified authentication using root keys
- ✅ **Permission-based Authorization**: RBAC with `api.*.create_key` or `api.{apiId}.create_key` permissions
- ✅ **Secure Key Hashing**: SHA-256 hashing for secure key storage
- ✅ **Input Validation**: Comprehensive validation of all input parameters

### Advanced Features
- ✅ **Usage Credits**: Support for initial credit allocation and consumption tracking
- ✅ **Rate Limiting**: Multiple named rate limits per key
- ✅ **Permissions & Roles**: Direct assignment of permissions and roles during creation
- ✅ **Metadata Support**: JSON metadata storage with keys
- ✅ **Expiration**: Optional key expiration timestamps
- ✅ **Enable/Disable**: Keys can be created in disabled state

### Not Implemented (Future Enhancements)
- ❌ **Credit Refill**: Automatic credit refill configuration (schema exists but not implemented)
- ❌ **Recoverable Keys**: Encrypted key storage for recovery (vault integration needed)

## Request Structure

```json
{
  "apiId": "api_123...",           // Required: Target API ID
  "name": "My API Key",            // Optional: Human-readable name
  "prefix": "prod",                // Optional: Key prefix (max 16 chars)
  "byteLength": 24,                // Optional: Key strength (16-255, default 16)
  "externalId": "user_123",        // Optional: External system identifier
  "meta": {"plan": "enterprise"},  // Optional: JSON metadata
  "expires": 1704067200000,        // Optional: Unix timestamp (ms)
  "enabled": true,                 // Optional: Initial state (default true)
  "permissions": ["read", "write"], // Optional: Direct permissions
  "roles": ["admin"],              // Optional: Role assignments
  "credits": {                     // Optional: Usage limits
    "remaining": 1000
  },
  "ratelimits": [{                 // Optional: Rate limiting
    "name": "requests",
    "limit": 100,
    "duration": 60000
  }]
}
```

## Response Structure

```json
{
  "meta": {
    "requestId": "req_123..."
  },
  "data": {
    "keyId": "key_123...",          // Database reference ID
    "key": "prod_AbC123..."         // Generated API key (returned once only)
  }
}
```

## Error Handling

### 400 Bad Request
- Missing or invalid `apiId`
- Invalid `byteLength` (must be 16-255)
- Invalid `prefix` (max 16 characters)
- Nonexistent permissions or roles
- Invalid metadata format
- Validation errors for any field

### 401 Unauthorized
- Missing Authorization header
- Invalid or malformed bearer token
- Nonexistent root key

### 403 Forbidden
- Insufficient permissions for API key creation
- Attempting to create keys for APIs in different workspaces

### 404 Not Found
- Nonexistent API ID
- API belongs to different workspace

## Implementation Details

### Database Operations
1. **API Validation**: Verify API exists and belongs to authorized workspace
2. **Permission/Role Resolution**: Validate and resolve permission/role names to IDs
3. **Key Insertion**: Atomic transaction for key creation with all related data
4. **Rate Limit Creation**: Individual rate limit records for each configured limit
5. **Permission/Role Assignment**: Link key to permissions and roles
6. **Audit Logging**: Record all creation and assignment operations

### Security Considerations
- Keys are hashed using SHA-256 before storage
- Only key hash is stored, not plaintext (except when `recoverable: true`)
- Workspace isolation prevents cross-tenant access
- Permission checks use OR logic for API-specific or wildcard permissions
- All operations are logged for audit purposes

### Key Generation Process
1. Generate random bytes using crypto/rand
2. Apply base58 encoding for URL-safe representation
3. Add optional prefix with underscore separator
4. Hash complete key for database storage
5. Extract start substring for indexing

## Test Coverage

### Success Tests (200)
- ✅ Basic key creation
- ✅ Key creation with optional fields
- ✅ Prefix validation and application
- ✅ Database persistence verification

### Validation Tests (400)
- ✅ Missing/empty/invalid apiId
- ✅ Invalid byteLength bounds
- ✅ Prefix length validation
- ✅ Nonexistent permissions/roles
- ✅ Empty or oversized permission/role names

### Authentication Tests (401)
- ✅ Missing authorization header
- ✅ Invalid bearer token format
- ✅ Nonexistent root keys

### Authorization Tests (403)
- ✅ No permissions
- ✅ Wrong permissions (e.g., read instead of create)
- ✅ API-specific permission mismatches
- ✅ Unrelated permissions

### Not Found Tests (404)
- ✅ Nonexistent API IDs
- ✅ Cross-workspace API access attempts

## Usage Examples

### Basic Key Creation
```bash
curl -X POST https://api.unkey.dev/v2/keys.createKey \
  -H "Authorization: Bearer unkey_123..." \
  -H "Content-Type: application/json" \
  -d '{"apiId": "api_123..."}'
```

### Advanced Key with All Features
```bash
curl -X POST https://api.unkey.dev/v2/keys.createKey \
  -H "Authorization: Bearer unkey_123..." \
  -H "Content-Type: application/json" \
  -d '{
    "apiId": "api_123...",
    "name": "Production Service Key",
    "prefix": "prod",
    "byteLength": 32,
    "externalId": "service_456",
    "meta": {"service": "payment", "version": "2.1"},
    "expires": 1735689600000,
    "permissions": ["payments.process", "customers.read"],
    "roles": ["service_account"],
    "credits": {"remaining": 10000},
    "ratelimits": [
      {"name": "requests", "limit": 1000, "duration": 3600000},
      {"name": "heavy_ops", "limit": 10, "duration": 60000}
    ]
  }'
```

## Files Structure

```
v2_keys_create_key/
├── handler.go          # Main endpoint implementation
├── 200_test.go         # Success scenario tests
├── 400_test.go         # Validation error tests
├── 401_test.go         # Authentication error tests
├── 403_test.go         # Authorization error tests
├── 404_test.go         # Not found error tests
└── README.md           # This documentation
```

## Integration

The endpoint is registered in `routes/register.go` and follows the same patterns as other v2 endpoints. It integrates with:

- **Authentication Service**: Root key verification
- **Permission Service**: RBAC authorization
- **Database Service**: Key and related data persistence
- **Audit Service**: Operation logging
- **Validation Framework**: Request validation

## Next Steps

To complete the full v2 keys API, the following endpoints still need implementation:

1. `v2/keys.deleteKey` - Delete API keys
2. `v2/keys.getKey` - Retrieve key information
3. `v2/keys.updateKey` - Update key properties
4. `v2/keys.updateRemaining` - Update usage credits
5. `v2/keys.verifyKey` - Verify and increment usage
6. `v2/keys.whoami` - Get key info without incrementing usage