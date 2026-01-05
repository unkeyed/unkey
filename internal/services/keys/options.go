package keys

import (
	"errors"

	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// VerifyOption represents a functional option for configuring key verification.
// Options can be combined to create complex validation scenarios.
type VerifyOption func(*verifyConfig) error

// verifyConfig holds the internal configuration for verification options.
type verifyConfig struct {
	ipWhitelist bool
	credits     *int32
	tags        []string
	permissions *rbac.PermissionQuery
	ratelimits  []openapi.KeysVerifyKeyRatelimit
}

// WithCredits validates that the key has sufficient usage credits and deducts the specified cost.
// The cost must be non-negative. If the key doesn't have enough credits, verification fails.
func WithCredits(cost int32) VerifyOption {
	return func(config *verifyConfig) error {
		if cost < 0 {
			return errors.New("cost cannot be negative")
		}

		config.credits = &cost
		return nil
	}
}

// WithIPWhitelist validates that the client IP address is in the key's IP whitelist.
// The client IP is extracted from the session. If no whitelist is configured, this check is skipped.
func WithIPWhitelist() VerifyOption {
	return func(config *verifyConfig) error {
		config.ipWhitelist = true
		return nil
	}
}

// WithPermissions validates that the key has the required RBAC permissions.
// The query specifies the action and resource that the key needs access to.
func WithPermissions(query rbac.PermissionQuery) VerifyOption {
	return func(config *verifyConfig) error {
		config.permissions = &query
		return nil
	}
}

// WithRateLimits validates the key against the specified rate limits.
// These limits are applied in addition to any auto-applied limits on the key or identity.
func WithRateLimits(limits []openapi.KeysVerifyKeyRatelimit) VerifyOption {
	return func(config *verifyConfig) error {
		config.ratelimits = limits
		return nil
	}
}

// WithTags adds given tags to the key verification.
func WithTags(tags []string) VerifyOption {
	return func(config *verifyConfig) error {
		config.tags = tags
		return nil
	}
}
