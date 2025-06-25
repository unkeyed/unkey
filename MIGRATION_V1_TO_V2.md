# Unkey API v1 to v2 Migration Guide

This guide will help you migrate from Unkey's v1 API to the new v2 API. The v2 API introduces significant improvements in structure, consistency, and functionality.

## Overview of Changes

### Major Structural Changes
- **Consistent Response Format**: All v2 responses follow a standardized format with `meta` and `data` fields
- **POST-based Operations**: Most operations now use POST requests instead of GET for better security and flexibility
- **Enhanced Error Handling**: More detailed error responses with validation details
- **New Resource Management**: Added support for identities, permissions, roles, and ratelimit overrides

## URL Structure Changes

### v1 → v2 Path Mapping

| v1 Path | v2 Path | Method Change |
|---------|---------|---------------|
| `GET /v1/keys.getKey` | `POST /v2/keys.getKey` | GET → POST |
| `POST /v1/keys.createKey` | `POST /v2/keys.createKey` | No change |
| `POST /v1/keys.deleteKey` | `POST /v2/keys.deleteKey` | No change |
| `POST /v1/keys.updateKey` | `POST /v2/keys.updateKey` | No change |
| `POST /v1/keys.verifyKey` | `POST /v2/keys.verifyKey` | No change |
| `POST /v1/keys.whoami` | `POST /v2/keys.whoami` | No change |
| `GET /v1/apis.getApi` | `POST /v2/apis.getApi` | GET → POST |
| `POST /v1/apis.createApi` | `POST /v2/apis.createApi` | No change |
| `GET /v1/apis.listKeys` | `POST /v2/apis.listKeys` | GET → POST |

## Response Format Changes

### v1 Response Format
```json
{
  "keyId": "key_123",
  "valid": true,
  "name": "Customer X"
}
```

### v2 Response Format
```json
{
  "meta": {
    "requestId": "req_1234"
  },
  "data": {
    "keyId": "key_123",
    "valid": true,
    "name": "Customer X"
  }
}
```

## Endpoint-Specific Migration

### 1. Key Operations

#### Get Key
**v1:**
```http
GET /v1/keys.getKey?keyId=key_123&decrypt=false
Authorization: Bearer <rootKey>
```

**v2:**
```http
POST /v2/keys.getKey
Authorization: Bearer <rootKey>
Content-Type: application/json

{
  "keyId": "key_123",
  "decrypt": false
}
```

#### Verify Key
**v1:**
```json
{
  "apiId": "api_123",
  "key": "sk_123"
}
```

**v2:**
```json
{
  "apiId": "api_123",
  "key": "sk_123",
  "permissions": {
    "type": "and",
    "permissions": ["read", "write"]
  },
  "ratelimits": [
    {
      "name": "requests",
      "cost": 1
    }
  ]
}
```

### 2. API Operations

#### Get API
**v1:**
```http
GET /v1/apis.getApi?apiId=api_123
Authorization: Bearer <rootKey>
```

**v2:**
```http
POST /v2/apis.getApi
Authorization: Bearer <rootKey>
Content-Type: application/json

{
  "apiId": "api_123"
}
```

#### List Keys
**v1:**
```http
GET /v1/apis.listKeys?apiId=api_123&limit=100
Authorization: Bearer <rootKey>
```

**v2:**
```http
POST /v2/apis.listKeys
Authorization: Bearer <rootKey>
Content-Type: application/json

{
  "apiId": "api_123",
  "limit": 100,
  "cursor": "optional_cursor_for_pagination"
}
```

### 3. New Features in v2

#### Permissions and Roles Management
v2 introduces comprehensive RBAC support:

```json
POST /v2/keys.addPermissions
{
  "keyId": "key_123",
  "permissions": [
    {
      "slug": "read_users",
      "create": true
    },
    {
      "id": "perm_456"
    }
  ]
}
```

#### Identity Management
New identity system for better user association:

```json
POST /v2/identities.createIdentity
{
  "externalId": "user_123",
  "meta": {
    "name": "John Doe",
    "email": "john@example.com"
  },
  "ratelimits": [
    {
      "name": "requests",
      "limit": 100,
      "duration": 60000
    }
  ]
}
```

#### Ratelimit Overrides
Dynamic ratelimit management:

```json
POST /v2/ratelimit.setOverride
{
  "namespaceName": "api_requests",
  "identifier": "user_123",
  "limit": 1000,
  "duration": 3600000
}
```

## Field Changes and Deprecations

### Deprecated Fields
- `ownerId` → Use `externalId` instead
- v1 ratelimit structure → Use new v2 ratelimit array format

### New Required Fields
- All responses now include `meta.requestId` for debugging
- Enhanced validation with detailed error responses

### Credits System Enhancement
**v1:**
```json
{
  "remaining": 100,
  "refill": {
    "interval": "monthly",
    "amount": 100
  }
}
```

**v2:**
```json
{
  "remaining": 100,
  "refill": {
    "interval": "monthly",
    "amount": 100,
    "refillDay": 1,
    "lastRefillAt": 1640995200000
  }
}
```

## Error Handling Changes

### v1 Error Format
```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid request",
    "requestId": "req_123"
  }
}
```

### v2 Error Format
```json
{
  "meta": {
    "requestId": "req_123"
  },
  "error": {
    "title": "Bad Request",
    "detail": "Validation failed",
    "status": 400,
    "type": "https://unkey.dev/docs/api-reference/errors/400",
    "errors": [
      {
        "location": "body.keyId",
        "message": "Required field missing",
        "fix": "Provide a valid keyId"
      }
    ]
  }
}
```

## Migration Steps

### 1. Update Base URLs
- No change needed - same base URL `https://api.unkey.dev`

### 2. Update Request Methods
- Change GET requests to POST for: `getKey`, `getApi`, `listKeys`
- Move query parameters to request body

### 3. Update Response Handling
```javascript
// v1 response handling
const response = await fetch('/v1/keys.getKey?keyId=key_123');
const keyData = await response.json();
console.log(keyData.name); // Direct access

// v2 response handling
const response = await fetch('/v2/keys.getKey', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ keyId: 'key_123' })
});
const result = await response.json();
console.log(result.data.name); // Access via data field
console.log(result.meta.requestId); // Access request ID for debugging
```

### 4. Update Error Handling
```javascript
// v1 error handling
if (!response.ok) {
  const error = await response.json();
  console.error(error.error.message);
}

// v2 error handling
if (!response.ok) {
  const error = await response.json();
  console.error(error.error.title);
  if (error.error.errors) {
    error.error.errors.forEach(err => {
      console.error(`${err.location}: ${err.message}`);
    });
  }
}
```

### 5. Leverage New Features
- Implement RBAC using the new permissions and roles system
- Use identities for better user management
- Set up ratelimit overrides for dynamic rate limiting

## Migration Checklist

- [ ] Update request methods from GET to POST where applicable
- [ ] Move query parameters to request body
- [ ] Update response handling to access `data` field
- [ ] Update error handling for new error format
- [ ] Test all existing functionality with v2 endpoints
- [ ] Implement new v2 features (RBAC, identities, etc.)
- [ ] Update client libraries/SDKs
- [ ] Monitor `meta.requestId` for debugging

## New v2 Endpoints

### Identity Management
- `POST /v2/identities.createIdentity`
- `POST /v2/identities.getIdentity`
- `POST /v2/identities.listIdentities`
- `POST /v2/identities.updateIdentity`
- `POST /v2/identities.deleteIdentity`

### Permissions Management
- `POST /v2/permissions.createPermission`
- `POST /v2/permissions.getPermission`
- `POST /v2/permissions.listPermissions`
- `POST /v2/permissions.deletePermission`

### Roles Management
- `POST /v2/permissions.createRole`
- `POST /v2/permissions.getRole`
- `POST /v2/permissions.listRoles`
- `POST /v2/permissions.deleteRole`

### Key Permissions & Roles
- `POST /v2/keys.addPermissions`
- `POST /v2/keys.removePermissions`
- `POST /v2/keys.setPermissions`
- `POST /v2/keys.addRoles`
- `POST /v2/keys.removeRoles`
- `POST /v2/keys.setRoles`

### Ratelimit Management
- `POST /v2/ratelimit.setOverride`
- `POST /v2/ratelimit.getOverride`
- `POST /v2/ratelimit.listOverrides`
- `POST /v2/ratelimit.deleteOverride`

## Backward Compatibility

- v1 endpoints remain available during the transition period
- No breaking changes to existing v1 functionality
- Gradual migration is supported

## Benefits of v2

1. **Consistent Structure**: All responses follow the same format
2. **Better Security**: POST requests for sensitive operations
3. **Enhanced RBAC**: Built-in permissions and roles system
4. **Identity Management**: Better user association and tracking
5. **Dynamic Ratelimiting**: Override limits on-the-fly
6. **Improved Debugging**: Request IDs in all responses
7. **Better Validation**: Detailed error messages with fix suggestions
8. **Enhanced Credits System**: More detailed refill information

## Timeline and Support

### Migration Timeline
- **Phase 1**: v2 API available alongside v1
- **Phase 2**: Deprecation warnings for v1 endpoints
- **Phase 3**: v1 endpoints sunset (timeline TBD)

### Getting Help
- Email: [support@unkey.dev](mailto:support@unkey.dev)
- Documentation: [unkey.dev/docs](https://unkey.dev/docs)
- Discord: [unkey.dev/discord](https://unkey.dev/discord)

## Next Steps

1. Review your current v1 API usage
2. Test v2 endpoints in your development environment
3. Update your client code to handle the new response format
4. Migrate critical endpoints first
5. Implement new v2 features like RBAC and identities
6. Complete migration and deprecate v1 usage

For additional support during migration, contact [support@unkey.dev](mailto:support@unkey.dev) with your specific use cases and requirements.