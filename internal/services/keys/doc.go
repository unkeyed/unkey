/*
Package keys implements a comprehensive key management and verification system for API keys
with support for rate limiting, usage tracking, permissions, and workspace isolation.

# Architecture

The keys service provides a unified interface for managing API keys throughout their lifecycle:

  - Key Creation: Secure generation of API keys with customizable prefixes and byte lengths
  - Key Verification: Multi-stage validation with configurable options for different use cases
  - Key Retrieval: Cached access to key metadata and authorization information
  - Root Key Management: Special handling for workspace-level administrative keys

# Key Verification System

The verification system uses a flexible, option-based approach that supports:

 1. Basic validation: existence, enabled status, expiration
 2. Usage limiting: credit-based consumption tracking
 3. Rate limiting: configurable time-window based limits
 4. Permission checking: RBAC-based authorization
 5. IP whitelisting: network-level access control
 6. Workspace isolation: multi-tenant security boundaries

# Usage

To create a new keys service:

	svc, err := keys.New(keys.Config{
	    DB:           database,
	    RateLimiter:  rateLimiter,
	    UsageLimiter: usageLimiter,
	    RBAC:         rbac,
	    Clickhouse:   clickhouse,
	    KeyCache:     keyCache,
	    WorkspaceCache: workspaceCache,
	})

To verify a key with rate limiting and permissions:

	key, err := svc.Get(ctx, session, rawKey)
	if err != nil {
	    return err
	}

	err = key.Verify(ctx,
	    keys.WithCredits(1),
	    keys.WithPermissions(rbac.PermissionQuery{
	        Action:   "read",
	        Resource: "api.key",
	    }),
	    keys.WithRateLimits([]openapi.KeysVerifyKeyRatelimit{
	        {Name: "requests", Limit: ptr.Int32(100), Duration: ptr.Int64(60000)},
	    }),
	)

	if !key.Valid {
	    // Handle validation failure based on key.Status
	}

# Key Statuses

The system defines comprehensive status codes for different validation outcomes:

  - VALID: Key passed all validation checks
  - NOT_FOUND: Key does not exist in the system
  - DISABLED: Key exists but is disabled
  - EXPIRED: Key has passed its expiration time
  - FORBIDDEN: Access denied (IP whitelist, etc.)
  - INSUFFICIENT_PERMISSIONS: RBAC validation failed
  - RATE_LIMITED: Rate limit exceeded
  - USAGE_EXCEEDED: Usage credit limit exceeded
  - WORKSPACE_DISABLED: Associated workspace is disabled
  - WORKSPACE_NOT_FOUND: Associated workspace does not exist

# Root Key Handling

Root keys receive special treatment with automatic fault error conversion:
- Validation failures are immediately converted to fault errors
- Used for workspace-level administrative operations
- Identified by the presence of ForWorkspaceID in the key metadata

# Thread Safety

The service is designed to be thread-safe and can handle concurrent requests across
multiple goroutines. All cache operations and state modifications are properly synchronized.

# Error Handling

The service provides structured error handling with:
- Fault errors for client-facing validation failures
- System errors for internal service problems
- Comprehensive error codes matching the OpenAPI specification
- Detailed logging for debugging and monitoring

# Performance Considerations

The service includes several performance optimizations:
- Multi-level caching (key cache, workspace cache)
- Stale-while-revalidate cache patterns
- Batched telemetry data collection
- Efficient database queries with proper indexing

See the KeyService interface and KeyVerifier struct for detailed documentation
of the API contract and available methods.
*/
package keys
