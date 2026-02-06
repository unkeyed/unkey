---
title: keys
description: "implements a comprehensive key management and verification system for API keys"
---

Package keys implements a comprehensive key management and verification system for API keys with support for rate limiting, usage tracking, permissions, and workspace isolation.

### Architecture

The keys service provides a unified interface for managing API keys throughout their lifecycle:

  - Key Creation: Secure generation of API keys with customizable prefixes and byte lengths
  - Key Verification: Multi-stage validation with configurable options for different use cases
  - Key Retrieval: Cached access to key metadata and authorization information
  - Root Key Management: Special handling for workspace-level administrative keys

### Key Verification System

The verification system uses a flexible, option-based approach that supports:

 1. Basic validation: existence, enabled status, expiration
 2. Usage limiting: credit-based consumption tracking
 3. Rate limiting: configurable time-window based limits
 4. Permission checking: RBAC-based authorization
 5. IP whitelisting: network-level access control
 6. Workspace isolation: multi-tenant security boundaries

### Usage

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

### Key Statuses

The system defines comprehensive status codes for different validation outcomes:

  - VALID: Key passed all validation checks
  - NOT\_FOUND: Key does not exist in the system
  - DISABLED: Key exists but is disabled
  - EXPIRED: Key has passed its expiration time
  - FORBIDDEN: Access denied (IP whitelist, etc.)
  - INSUFFICIENT\_PERMISSIONS: RBAC validation failed
  - RATE\_LIMITED: Rate limit exceeded
  - USAGE\_EXCEEDED: Usage credit limit exceeded
  - WORKSPACE\_DISABLED: Associated workspace is disabled
  - WORKSPACE\_NOT\_FOUND: Associated workspace does not exist

### Root Key Handling

Root keys receive special treatment with automatic fault error conversion: - Validation failures are immediately converted to fault errors - Used for workspace-level administrative operations - Identified by the presence of ForWorkspaceID in the key metadata

### Thread Safety

The service is designed to be thread-safe and can handle concurrent requests across multiple goroutines. All cache operations and state modifications are properly synchronized.

### Error Handling

The service provides structured error handling with: - Fault errors for client-facing validation failures - System errors for internal service problems - Comprehensive error codes matching the OpenAPI specification - Detailed logging for debugging and monitoring

### Performance Considerations

The service includes several performance optimizations: - Multi-level caching (key cache, workspace cache) - Stale-while-revalidate cache patterns - Batched telemetry data collection - Efficient database queries with proper indexing

See the KeyService interface and KeyVerifier struct for detailed documentation of the API contract and available methods.

## Variables

```go
var emptyLog = func() {}
```


## Types

### type Config

```go
type Config struct {
	DB           db.Database           // Database connection
	RateLimiter  ratelimit.Service     // Rate limiting service
	RBAC         *rbac.RBAC            // Role-based access control
	Clickhouse   clickhouse.ClickHouse // Clickhouse for telemetry
	Region       string                // Geographic region identifier
	UsageLimiter usagelimiter.Service  // Redis Counter for usage limiting

	KeyCache cache.Cache[string, db.CachedKeyData] // Cache for key lookups with pre-parsed data
}
```

Config holds the configuration for creating a new keys service instance.

### type CreateKeyRequest

```go
type CreateKeyRequest struct {
	Prefix     string // Optional prefix to prepend to the key (e.g., "test_", "prod_")
	ByteLength int    // Length of the random bytes to generate (16-255)
}
```

CreateKeyRequest specifies the parameters for creating a new API key.

### type CreateKeyResponse

```go
type CreateKeyResponse struct {
	Key   string // The complete plaintext key (prefix + encoded random bytes)
	Hash  string // SHA-256 hash of the key for secure storage
	Start string // The start of the key for indexing and display purposes
}
```

CreateKeyResponse contains the generated key and its metadata.

### type KeyService

```go
type KeyService interface {
	// Get retrieves a sha256 hashed key and returns a KeyVerifier for validation
	Get(ctx context.Context, sess *zen.Session, hash string) (*KeyVerifier, func(), error)

	// GetRootKey retrieves and validates a root key from the session
	GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, func(), error)

	// GetMigrated retrieves a key using rawKey and migrationID
	// If migration is pending, it performs on-demand migration and returns a KeyVerifier for further validation.
	GetMigrated(ctx context.Context, sess *zen.Session, rawKey string, migrationID string) (*KeyVerifier, func(), error)

	// CreateKey generates a new secure API key
	CreateKey(ctx context.Context, req CreateKeyRequest) (CreateKeyResponse, error)
}
```

KeyService defines the interface for key management operations. It provides methods for key creation, retrieval, and validation.

### type KeyStatus

```go
type KeyStatus string
```

KeyStatus represents the validation status of a key after verification. Each status indicates a specific validation outcome that can be used to determine the appropriate response and error handling.

```go
const (
	StatusValid                   KeyStatus = "VALID"
	StatusNotFound                KeyStatus = "NOT_FOUND"
	StatusDisabled                KeyStatus = "DISABLED"
	StatusExpired                 KeyStatus = "EXPIRED"
	StatusForbidden               KeyStatus = "FORBIDDEN"
	StatusInsufficientPermissions KeyStatus = "INSUFFICIENT_PERMISSIONS"
	StatusRateLimited             KeyStatus = "RATE_LIMITED"
	StatusUsageExceeded           KeyStatus = "USAGE_EXCEEDED"
	StatusWorkspaceDisabled       KeyStatus = "WORKSPACE_DISABLED"
	StatusWorkspaceNotFound       KeyStatus = "WORKSPACE_NOT_FOUND"
)
```

### type KeyVerifier

```go
type KeyVerifier struct {
	Key                   db.FindKeyForVerificationRow // The key data from the database
	Roles                 []string                     // RBAC roles assigned to this key
	Permissions           []string                     // RBAC permissions assigned to this key
	Status                KeyStatus                    // The current validation status
	AuthorizedWorkspaceID string                       // The workspace ID this key is authorized for

	ratelimitConfigs map[string]db.KeyFindForVerificationRatelimit // Rate limits configured for this key (name -> config)
	RatelimitResults map[string]RatelimitConfigAndResult           // Combined config and results for rate limits (name -> config+result)

	parsedIPWhitelist map[string]struct{} // Pre-parsed IP whitelist for O(1) lookup
	isRootKey         bool                // Whether this is a root key (special handling)

	message string   // Internal message for validation failures
	tags    []string // Tags associated with this verification

	session *zen.Session // The current request session
	region  string       // Geographic region identifier

	// Tracking
	startTime    time.Time // Time when verification started
	spentCredits int64     // Credits spent during verification

	// Services
	rateLimiter  ratelimit.Service     // Rate limiting service
	usageLimiter usagelimiter.Service  // Usage limiting service
	rBAC         *rbac.RBAC            // Role-based access control service
	clickhouse   clickhouse.ClickHouse // Clickhouse for telemetry
}
```

KeyVerifier represents a key that has been loaded from the database and is ready for verification. It contains all the necessary information and services to perform various validation checks.

#### func (KeyVerifier) GetRatelimitConfigs

```go
func (k *KeyVerifier) GetRatelimitConfigs() map[string]db.KeyFindForVerificationRatelimit
```

GetRatelimitConfigs returns the rate limit configurations

#### func (KeyVerifier) HasAnyPermission

```go
func (k *KeyVerifier) HasAnyPermission(resourceType rbac.ResourceType, action rbac.ActionType) bool
```

HasAnyPermission checks if the key has any permission matching the given action. It returns true if the key has at least one permission that ends with the specified action for the given resource type (e.g., checking for any "api.\*.verify\_key" or "api.{apiId}.verify\_key").

#### func (KeyVerifier) ToFault

```go
func (k *KeyVerifier) ToFault() error
```

ToFault converts the verification result to an appropriate fault error. This method should only be called when k.Valid is false. It provides structured error information that matches the API specification.

#### func (KeyVerifier) ToOpenAPIStatus

```go
func (k *KeyVerifier) ToOpenAPIStatus() openapi.V2KeysVerifyKeyResponseDataCode
```

ToOpenAPIStatus converts our internal KeyStatus to the OpenAPI response status type. This mapping ensures consistency between internal validation and external API responses.

#### func (KeyVerifier) Verify

```go
func (k *KeyVerifier) Verify(ctx context.Context, opts ...VerifyOption) error
```

Verify performs key verification with the given options. For root keys: returns fault errors for validation failures. For normal keys: returns error only for system problems, check k.Valid and k.Status for validation results.

#### func (KeyVerifier) VerifyRootKey

```go
func (k *KeyVerifier) VerifyRootKey(ctx context.Context, opts ...VerifyOption) error
```

### type RatelimitConfigAndResult

```go
type RatelimitConfigAndResult struct {
	ID         string
	Cost       int64
	Name       string
	Duration   time.Duration
	Limit      int64
	AutoApply  bool
	Identifier string                       // The identifier to use for this rate limit
	Response   *ratelimit.RatelimitResponse // nil until rate limit is checked
}
```

RatelimitConfigAndResult holds both the configuration and result for a rate limit

### type VerifyOption

```go
type VerifyOption func(*verifyConfig) error
```

VerifyOption represents a functional option for configuring key verification. Options can be combined to create complex validation scenarios.

#### func WithCredits

```go
func WithCredits(cost int32) VerifyOption
```

WithCredits validates that the key has sufficient usage credits and deducts the specified cost. The cost must be non-negative. If the key doesn't have enough credits, verification fails.

#### func WithIPWhitelist

```go
func WithIPWhitelist() VerifyOption
```

WithIPWhitelist validates that the client IP address is in the key's IP whitelist. The client IP is extracted from the session. If no whitelist is configured, this check is skipped.

#### func WithPermissions

```go
func WithPermissions(query rbac.PermissionQuery) VerifyOption
```

WithPermissions validates that the key has the required RBAC permissions. The query specifies the action and resource that the key needs access to.

#### func WithRateLimits

```go
func WithRateLimits(limits []openapi.KeysVerifyKeyRatelimit) VerifyOption
```

WithRateLimits validates the key against the specified rate limits. These limits are applied in addition to any auto-applied limits on the key or identity.

#### func WithTags

```go
func WithTags(tags []string) VerifyOption
```

WithTags adds given tags to the key verification.

### type VerifyResponse

```go
type VerifyResponse struct {
	AuthorizedWorkspaceID string // The workspace ID that the key is authorized for
	KeyID                 string // The unique identifier of the key
}
```

VerifyResponse contains the result of a successful key verification.

