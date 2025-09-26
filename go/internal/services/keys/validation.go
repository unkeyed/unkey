package keys

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"slices"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

// withCredits validates that the key has sufficient usage credits and deducts the specified cost.
// It updates the key's remaining request count and marks the key as invalid if the limit is exceeded.
func (k *KeyVerifier) withCredits(ctx context.Context, cost int32) error {
	ctx, span := tracing.Start(ctx, "verify.withCredits")
	defer span.End()

	if k.Status != StatusValid {
		return nil
	}

	// Key has unlimited requests if set to NULL
	if !k.Key.RemainingRequests.Valid {
		return nil
	}

	usage, err := k.usageLimiter.Limit(ctx, usagelimiter.UsageRequest{
		KeyId: k.Key.ID,
		Cost:  cost,
	})
	if err != nil {
		return err
	}

	// Always update remaining requests with the accurate count from the usageLimiter
	k.Key.RemainingRequests = sql.NullInt32{Int32: usage.Remaining, Valid: true}
	if !usage.Valid {
		k.setInvalid(StatusUsageExceeded, "Key usage limit exceeded.")
	}

	return nil
}

// withIPWhitelist validates that the client IP address is in the key's IP whitelist.
// If no whitelist is configured, this validation is skipped.
func (k *KeyVerifier) withIPWhitelist() error {
	if k.Status != StatusValid {
		return nil
	}

	if !k.Key.IpWhitelist.Valid {
		return nil
	}

	clientIP := k.session.Location()
	if clientIP == "" {
		k.Status = StatusForbidden
		k.message = "client IP is required for IP whitelist validation"
		return nil
	}

	allowedIPs := strings.Split(k.Key.IpWhitelist.String, ",")
	for i, ip := range allowedIPs {
		allowedIPs[i] = strings.TrimSpace(ip)
	}

	if !slices.Contains(allowedIPs, clientIP) {
		k.setInvalid(StatusForbidden, fmt.Sprintf("client IP %s is not in the whitelist", clientIP))
	}

	return nil
}

// withPermissions validates that the key has the required RBAC permissions.
// It uses the configured RBAC system to evaluate the permission query against the key's permissions.
func (k *KeyVerifier) withPermissions(ctx context.Context, query rbac.PermissionQuery) error {
	ctx, span := tracing.Start(ctx, "verify.withPermissions")
	defer span.End()

	if k.Status != StatusValid {
		return nil
	}

	allowed, err := k.rBAC.EvaluatePermissions(query, k.Permissions)
	if err != nil {
		return err
	}

	if !allowed.Valid {
		k.setInvalid(StatusInsufficientPermissions, allowed.Message)
	}

	return nil
}

// withRateLimits validates the key against both auto-applied and specified rate limits.
// Auto-applied limits come from the key or identity configuration, while specified limits
// are provided at verification time. All limits must pass for the key to be valid.
func (k *KeyVerifier) withRateLimits(ctx context.Context, specifiedLimits []openapi.KeysVerifyKeyRatelimit) error {
	// nolint:ineffassign
	ctx, span := tracing.Start(ctx, "verify.withRateLimits")
	defer span.End()

	if k.Status != StatusValid {
		return nil
	}

	ratelimitsToCheck := make(map[string]RatelimitConfigAndResult)
	for name, rl := range k.ratelimitConfigs {
		if rl.AutoApply == 0 {
			continue
		}

		identifier := k.Key.ID
		if rl.IdentityID != "" {
			identifier = rl.IdentityID
		}

		ratelimitsToCheck[name] = RatelimitConfigAndResult{
			ID:         rl.ID,
			Cost:       1,
			Name:       rl.Name,
			Duration:   time.Duration(rl.Duration) * time.Millisecond,
			Limit:      int64(rl.Limit),
			AutoApply:  rl.AutoApply == 1,
			Identifier: identifier,
			Response:   nil,
		}
	}

	for _, rl := range specifiedLimits {
		// Custom limits are always applied on a key level
		if rl.Limit != nil && rl.Duration != nil {
			ratelimitsToCheck[rl.Name] = RatelimitConfigAndResult{
				Cost:       int64(ptr.SafeDeref(rl.Cost, 1)),
				Name:       rl.Name,
				Duration:   time.Duration(*rl.Duration) * time.Millisecond,
				Limit:      int64(*rl.Limit),
				AutoApply:  false,
				Identifier: k.Key.ID,
				Response:   nil,
				ID:         "", // Doesn't exist and is custom so no ID
			}

			continue
		}

		dbRl, exists := k.ratelimitConfigs[rl.Name]
		if !exists {
			errorMsg := "ratelimit '%s' was requested but does not exist for key '%s'"
			if k.Key.IdentityID.Valid {
				errorMsg += " nor identity: '%s' external ID: '%s'"
			} else {
				errorMsg += " and there is no identity connected."
			}

			errorMsg = fmt.Sprintf(errorMsg, rl.Name, k.Key.ID, k.Key.IdentityID.String, k.Key.ExternalID.String)
			return fault.New("invalid ratelimit requested",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Public(errorMsg),
			)
		}

		identifier := k.Key.ID
		if dbRl.IdentityID != "" {
			identifier = dbRl.IdentityID
		}

		ratelimitsToCheck[rl.Name] = RatelimitConfigAndResult{
			Name:       dbRl.Name,
			Duration:   time.Duration(dbRl.Duration) * time.Millisecond,
			Cost:       int64(ptr.SafeDeref(rl.Cost, 1)),
			Limit:      int64(dbRl.Limit),
			AutoApply:  dbRl.AutoApply == 1,
			Identifier: identifier,
			Response:   nil,
			ID:         dbRl.ID,
		}
	}

	if len(ratelimitsToCheck) == 0 {
		return nil
	}

	for name, config := range ratelimitsToCheck {
		response, err := k.rateLimiter.Ratelimit(ctx, ratelimit.RatelimitRequest{
			Identifier: config.Identifier, // Use the pre-determined identifier
			Limit:      config.Limit,
			Duration:   config.Duration,
			Cost:       config.Cost,
			Time:       time.Now(),
		})
		if err != nil {
			k.logger.Error("Failed to ratelimit",
				"key_id", k.Key.ID,
				"Identifier", config.Identifier,
				"error", err.Error(),
			)

			// We will just allow the request to proceed, but log the error
			return nil
		}

		config.Response = &response
		ratelimitsToCheck[name] = config

		// If rate limit exceeded, stop processing
		if !response.Success {
			k.setInvalid(StatusRateLimited, fmt.Sprintf("key exceeded rate limit %s", name))
			break
		}
	}

	// Store the final results
	k.RatelimitResults = ratelimitsToCheck

	return nil
}
