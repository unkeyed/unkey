# Unkey v2 API Endpoints

This document provides a comprehensive list of all endpoints in the Unkey v2 API, organized by functional category. Each endpoint includes its HTTP method, path, and primary purpose.

## APIs Management

APIs serve as organizational containers for keys, providing isolation between environments and services.

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/apis` | `createApi` | Create a new API namespace for organizing keys |
| `POST` | `/v2/apis/get` | `getApi` | Retrieve information about an API namespace |
| `POST` | `/v2/apis/delete` | `deleteApi` | Delete an API and invalidate all its associated keys |
| `POST` | `/v2/apis/listKeys` | `listKeys` | List all keys associated with a specific API |

## Key Management

Core functionality for creating, managing, and verifying API keys.

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/keys` | `createKey` | Create a new API key with customizable security features |
| `POST` | `/v2/keys/verify` | `verifyKey` | Verify an API key's validity and permissions |
| `POST` | `/v2/keys/get` | `getKey` | Retrieve information about a specific key |
| `POST` | `/v2/keys/update` | `updateKey` | Update key properties like name, metadata, or expiration |
| `POST` | `/v2/keys/delete` | `deleteKey` | Delete a key and invalidate it permanently |
| `POST` | `/v2/keys/updateCredits` | `updateCredits` | Modify the credit balance for a specific key |

## Permission Management

Manage fine-grained access control through permissions and roles.

### Permissions

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/permissions` | `createPermission` | Create a new permission for access control |
| `POST` | `/v2/permissions/get` | `getPermission` | Retrieve details about a specific permission |
| `POST` | `/v2/permissions/list` | `listPermissions` | List all permissions in the workspace |
| `POST` | `/v2/permissions/delete` | `deletePermission` | Delete a permission from the system |

### Roles

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/permissions/roles` | `createRole` | Create a new role containing multiple permissions |
| `POST` | `/v2/permissions/roles/get` | `getRole` | Retrieve details about a specific role |
| `POST` | `/v2/permissions/roles/list` | `listRoles` | List all roles in the workspace |
| `POST` | `/v2/permissions/roles/delete` | `deleteRole` | Delete a role from the system |

### Key Permission Operations

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/keys/addPermissions` | `addPermissions` | Add permissions to an existing key |
| `POST` | `/v2/keys/removePermissions` | `removePermissions` | Remove permissions from a key |
| `POST` | `/v2/keys/setPermissions` | `setPermissions` | Replace all permissions on a key |
| `POST` | `/v2/keys/addRoles` | `addRoles` | Add roles to an existing key |
| `POST` | `/v2/keys/removeRoles` | `removeRoles` | Remove roles from a key |
| `POST` | `/v2/keys/setRoles` | `setRoles` | Replace all roles on a key |

## Identity Management

Manage identities for organizing keys and access patterns.

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/identities` | `createIdentity` | Create a new identity for grouping keys |
| `POST` | `/v2/identities/get` | `getIdentity` | Retrieve information about a specific identity |
| `POST` | `/v2/identities/list` | `listIdentities` | List all identities in the workspace |
| `POST` | `/v2/identities/update` | `updateIdentity` | Update identity properties and metadata |
| `POST` | `/v2/identities/delete` | `deleteIdentity` | Delete an identity from the system |

## Rate Limiting

Flexible rate limiting system for any identifier or resource.

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/ratelimit` | `ratelimit.limit` | Apply rate limiting to any identifier |

### Rate Limit Overrides

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `POST` | `/v2/ratelimit/overrides` | `setOverride` | Create or update custom rate limit for specific identifier |
| `POST` | `/v2/ratelimit/overrides/get` | `getOverride` | Retrieve rate limit override for an identifier |
| `POST` | `/v2/ratelimit/overrides/list` | `listOverrides` | List all rate limit overrides |
| `POST` | `/v2/ratelimit/overrides/delete` | `deleteOverride` | Remove rate limit override for an identifier |

## System Health

| Method | Endpoint | Operation ID | Summary |
|--------|----------|--------------|---------|
| `GET` | `/v2/liveness` | `liveness` | Check service health and availability |

## Summary

The Unkey v2 API provides **36 endpoints** across **6 main categories**:

- **APIs Management**: 4 endpoints
- **Key Management**: 6 endpoints  
- **Permission Management**: 14 endpoints (4 permissions + 4 roles + 6 key operations)
- **Identity Management**: 5 endpoints
- **Rate Limiting**: 6 endpoints (1 core + 5 override management)
- **System Health**: 1 endpoint

### Key Characteristics

- **Consistent Design**: All endpoints use POST method with JSON request bodies (except liveness check)
- **Security**: All endpoints require root key authentication except liveness
- **Comprehensive Error Handling**: Standard HTTP status codes with detailed error responses
- **Rich Examples**: Each endpoint includes multiple request/response examples
- **Detailed Documentation**: Extensive descriptions with use cases and best practices

### Authentication

All endpoints require Bearer token authentication using a root key in the `Authorization` header, except:
- `GET /v2/liveness` - No authentication required

### Content Type

All POST endpoints:
- **Request**: `application/json`
- **Response**: `application/json`