package keys

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"golang.org/x/exp/slices"
)

type KeyVerifier struct {
	Key         db.FindKeyForVerificationRow
	Ratelimits  []db.KeyFindForVerificationRatelimit
	Roles       []string
	Permissions []string

	Valid                 bool
	Status                openapi.KeysVerifyKeyResponseDataCode
	AuthorizedWorkspaceID string
	IsRootKey             bool

	session *zen.Session

	rateLimiter  ratelimit.Service
	usageLimiter usagelimiter.Service
	rBAC         *rbac.RBAC
	clickhouse   clickhouse.ClickHouse

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

// Verify performs key verification with the given options
func (k *KeyVerifier) Verify(ctx context.Context, opts ...VerifyOption) error {
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

	return k.toError()
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
		k.Status = openapi.USAGEEXCEEDED
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

	clientIP := k.session.Location()
	if clientIP == "" {
		return errors.New("client IP is required for IP whitelist validation")
	}

	allowedIPs := strings.Split(k.Key.IpWhitelist.String, ",")
	if !slices.Contains(allowedIPs, clientIP) {
		k.Valid = false
		k.Status = openapi.FORBIDDEN
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
		k.Status = openapi.FORBIDDEN
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

// ToError converts the verification result to an appropriate fault error for root keys
func (k *KeyVerifier) toError() error {
	if k.Valid {
		return nil
	}

	// For root keys, return structured fault errors
	if k.IsRootKey {
		switch k.Status {
		case openapi.FORBIDDEN:
			// Use specific permission message if available
			publicMsg := "Insufficient permissions to access this resource."
			internalMsg := "key verification failed permission check"
			if k.message != "" {
				publicMsg = k.message
				internalMsg = k.message
			}
			return fault.New("insufficient permissions",
				fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.Internal(internalMsg),
				fault.Public(publicMsg),
			)
		case openapi.DISABLED:
			// Use specific message if available
			publicMsg := "The key is disabled."
			internalMsg := "key is disabled"
			if k.message != "" {
				publicMsg = k.message
				internalMsg = k.message
			}
			return fault.New("key is disabled",
				fault.Code(codes.Auth.Authorization.KeyDisabled.URN()),
				fault.Internal(internalMsg),
				fault.Public(publicMsg),
			)
		case openapi.NOTFOUND:
			return fault.New("key not found",
				fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
				fault.Internal("key does not exist"),
				fault.Public("We could not find the requested key."),
			)
		case openapi.EXPIRED:
			// Use specific message if available
			publicMsg := "The key has expired."
			internalMsg := "key has expired"
			if k.message != "" {
				publicMsg = k.message
				internalMsg = k.message
			}
			return fault.New("key has expired",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal(internalMsg),
				fault.Public(publicMsg),
			)
		case openapi.USAGEEXCEEDED:
			// Use specific message if available
			publicMsg := "Key usage limit exceeded."
			internalMsg := "key usage limit exceeded"
			if k.message != "" {
				publicMsg = k.message
				internalMsg = k.message
			}
			return fault.New("key usage limit exceeded",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal(internalMsg),
				fault.Public(publicMsg),
			)
		default:
			return fault.New("key verification failed",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal("key verification failed with unknown status"),
				fault.Public("Key verification failed."),
			)
		}
	}

	// For normal keys, return nil (no error) - they handle invalid state internally
	return nil
}
