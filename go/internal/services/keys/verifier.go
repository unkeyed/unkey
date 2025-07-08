package keys

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"golang.org/x/exp/slices"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
)

// KeyStatus represents the validation status of a key
type KeyStatus string

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

type KeyVerifier struct {
	Key         db.FindKeyForVerificationRow
	Ratelimits  []db.KeyFindForVerificationRatelimit
	Roles       []string
	Permissions []string

	Valid                 bool
	Status                KeyStatus
	AuthorizedWorkspaceID string
	IsRootKey             bool

	session *zen.Session

	rateLimiter  ratelimit.Service
	usageLimiter usagelimiter.Service
	rBAC         *rbac.RBAC
	clickhouse   clickhouse.ClickHouse
	logger       logging.Logger

	// Private field to store specific error messages from failed validations
	message string
}

// VerifyOption represents a verification option
type VerifyOption func(*verifyConfig) error

// verifyConfig holds the configuration for verification
type verifyConfig struct {
	credits     *int32
	permissions *rbac.PermissionQuery
	rateLimits  []string
	ipWhitelist bool
}

// WithCredits validates that the key has sufficient credits and deducts the cost
func WithCredits(cost int32) VerifyOption {
	return func(config *verifyConfig) error {
		if cost < 0 {
			return errors.New("cost cannot be negative")
		}
		config.credits = &cost
		return nil
	}
}

// WithIPWhitelist validates that the client IP is in the key's whitelist
func WithIPWhitelist() VerifyOption {
	return func(config *verifyConfig) error {
		config.ipWhitelist = true
		return nil
	}
}

// WithPermissions validates that the key has the required permissions
func WithPermissions(query rbac.PermissionQuery) VerifyOption {
	return func(config *verifyConfig) error {
		config.permissions = &query
		return nil
	}
}

// WithRateLimits validates against the specified rate limits
func WithRateLimits(limits []string) VerifyOption {
	return func(config *verifyConfig) error {
		config.rateLimits = limits
		return nil
	}
}

// Verify performs key verification with the given options.
// For root keys: returns fault errors for validation failures.
// For normal keys: returns error only for system problems, check k.Valid and k.Status for validation results.
func (k *KeyVerifier) Verify(ctx context.Context, opts ...VerifyOption) error {
	// Skip verification if key is already invalid
	if !k.Valid {
		// For root keys, auto-return validation failures as fault errors
		if k.IsRootKey {
			return k.ToFault()
		}
		return nil
	}

	config := &verifyConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return err
		}
	}

	if config.credits != nil {
		if err := k.withCredits(ctx, *config.credits); err != nil {
			return err
		}
	}

	if config.ipWhitelist {
		if err := k.withIPWhitelist(ctx); err != nil {
			return err
		}
	}

	if config.permissions != nil {
		if err := k.withPermissions(ctx, *config.permissions); err != nil {
			return err
		}
	}

	if len(config.rateLimits) > 0 {
		if err := k.withRateLimits(ctx, config.rateLimits); err != nil {
			return err
		}
	}

	// Buffer telemetry data
	k.clickhouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
		RequestID:   k.session.RequestID(),
		WorkspaceID: k.session.AuthorizedWorkspaceID(),
		Time:        time.Now().UnixMilli(),
		Region:      "",
		Outcome:     string(k.Status),
		KeySpaceID:  k.Key.KeyAuthID,
		KeyID:       k.Key.ID,
		IdentityID:  k.Key.IdentityID.String,
		Tags:        []string{},
	})

	// For root keys, auto-return validation failures as fault errors
	if k.IsRootKey && !k.Valid {
		return k.ToFault()
	}

	return nil
}

// Internal validation methods (private)

func (k *KeyVerifier) withCredits(ctx context.Context, cost int32) error {
	if !k.Valid {
		return nil
	}

	usage, err := k.usageLimiter.Limit(ctx, usagelimiter.UsageRequest{
		KeyId: k.Key.ID,
		Cost:  cost,
	})
	if err != nil {
		return err
	}

	k.Key.RemainingRequests = sql.NullInt32{Int32: usage.Remaining, Valid: true}
	if !usage.Valid {
		k.Valid = false
		k.Status = StatusUsageExceeded
		k.message = "Key usage limit exceeded."
	}
	return nil
}

func (k *KeyVerifier) withIPWhitelist(ctx context.Context) error {
	if !k.Valid {
		return nil
	}

	if !k.Key.IpWhitelist.Valid {
		return nil
	}

	// Get client IP from headers as per TypeScript implementation
	// Checks "True-Client-IP" then "CF-Connecting-IP"
	clientIP := k.session.Location()
	if clientIP == "" {
		k.Valid = false
		k.Status = StatusForbidden
		k.message = "client IP is required for IP whitelist validation"
		return nil
	}

	allowedIPs := strings.Split(k.Key.IpWhitelist.String, ",")
	// Trim whitespace from each IP as per TypeScript implementation
	for i, ip := range allowedIPs {
		allowedIPs[i] = strings.TrimSpace(ip)
	}

	if !slices.Contains(allowedIPs, clientIP) {
		k.Valid = false
		k.Status = StatusForbidden
		k.message = fmt.Sprintf("client IP %s is not in the whitelist", clientIP)
	}

	return nil
}

func (k *KeyVerifier) withPermissions(ctx context.Context, query rbac.PermissionQuery) error {
	if !k.Valid {
		return nil
	}

	allowed, err := k.rBAC.EvaluatePermissions(query, k.Permissions)
	if err != nil {
		return err
	}

	if !allowed.Valid {
		k.Valid = false
		k.Status = StatusInsufficientPermissions
		k.message = allowed.Message
	}

	return nil
}

func (k *KeyVerifier) withRateLimits(ctx context.Context, specifiedLimits []string) error {
	if !k.Valid {
		return nil
	}

	// Implementation would go here
	return nil
}

// ToFault converts the verification result to an appropriate fault error.
// This matches the error messages from both TypeScript service and old_keys service.
func (k *KeyVerifier) ToFault() error {
	if k.Valid {
		return nil
	}

	switch k.Status {
	case StatusNotFound:
		return fault.New("key does not exist",
			fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
			fault.Internal("key does not exist"),
			fault.Public("We could not find the requested key."),
		)
	case StatusDisabled:
		message := k.message
		if message == "" {
			message = "the key is disabled"
		}
		return fault.New("key is disabled",
			fault.Code(codes.Auth.Authorization.KeyDisabled.URN()),
			fault.Internal(message),
			fault.Public("The key is disabled."),
		)
	case StatusExpired:
		message := k.message
		if message == "" {
			message = "the key has expired"
		}
		return fault.New("key has expired",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusWorkspaceDisabled:
		return fault.New("workspace is disabled",
			fault.Code(codes.Auth.Authorization.WorkspaceDisabled.URN()),
			fault.Internal("workspace disabled"),
			fault.Public("The workspace is disabled."),
		)
	case StatusWorkspaceNotFound:
		return fault.New("workspace not found",
			fault.Code(codes.Data.Workspace.NotFound.URN()),
			fault.Internal("workspace disabled"),
			fault.Public("The requested workspace does not exist."),
		)
	case StatusForbidden:
		message := k.message
		if message == "" {
			message = "Forbidden"
		}
		return fault.New("forbidden",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusInsufficientPermissions:
		message := k.message
		if message == "" {
			message = "Insufficient permissions to access this resource."
		}
		return fault.New("insufficient permissions",
			fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusUsageExceeded:
		message := k.message
		if message == "" {
			message = "Key usage limit exceeded."
		}
		return fault.New("key usage limit exceeded",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusRateLimited:
		message := k.message
		if message == "" {
			message = "Rate limit exceeded"
		}
		return fault.New("rate limit exceeded",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	default:
		return fault.New("key verification failed",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("key verification failed with unknown status"),
			fault.Public("Key verification failed."),
		)
	}
}

// ToOpenAPIStatus converts our KeyStatus to the OpenAPI status type
func (k *KeyVerifier) ToOpenAPIStatus() openapi.KeysVerifyKeyResponseDataCode {
	switch k.Status {
	case StatusValid:
		return openapi.VALID
	case StatusNotFound:
		return openapi.NOTFOUND
	case StatusDisabled:
		return openapi.DISABLED
	case StatusExpired:
		return openapi.EXPIRED
	case StatusForbidden:
		return openapi.FORBIDDEN
	case StatusInsufficientPermissions:
		return openapi.INSUFFICIENTPERMISSIONS
	case StatusUsageExceeded:
		return openapi.USAGEEXCEEDED
	case StatusRateLimited:
		// OpenAPI doesn't have RATE_LIMITED, so use FORBIDDEN
		return openapi.FORBIDDEN
	case StatusWorkspaceDisabled:
		// OpenAPI doesn't have WORKSPACE_DISABLED, so use FORBIDDEN
		return openapi.FORBIDDEN
	default:
		return openapi.FORBIDDEN
	}
}
