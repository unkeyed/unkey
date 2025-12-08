package keys

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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
	} else {
		// Only track spent credits if they were actually spent (usage was valid)
		k.spentCredits = int64(cost)
	}

	return nil
}

// withIPWhitelist validates that the client IP address is in the key's IP whitelist.
// If no whitelist is configured, this validation is skipped.
func (k *KeyVerifier) withIPWhitelist() error {
	if k.Status != StatusValid {
		return nil
	}

	if len(k.parsedIPWhitelist) == 0 {
		return nil
	}

	clientIP := k.session.Location()
	if clientIP == "" {
		k.Status = StatusForbidden
		k.message = "client IP is required for IP whitelist validation"
		return nil
	}

	if _, ok := k.parsedIPWhitelist[clientIP]; !ok {
		k.setInvalid(StatusForbidden, fmt.Sprintf("client IP %s is not in the whitelist", clientIP))
	}

	return nil
}

// HasAnyPermission checks if the key has any permission matching the given action.
// It returns true if the key has at least one permission that ends with the specified action
// for the given resource type (e.g., checking for any "api.*.verify_key" or "api.{apiId}.verify_key").
func (k *KeyVerifier) HasAnyPermission(resourceType rbac.ResourceType, action rbac.ActionType) bool {
	prefix := string(resourceType) + "."
	suffix := "." + string(action)

	for _, perm := range k.Permissions {
		if strings.HasPrefix(perm, prefix) && strings.HasSuffix(perm, suffix) {
			return true
		}
	}

	return false
}

// withPermissions validates that the key has the required RBAC permissions.
// It uses the configured RBAC system to evaluate the permission query against the key's permissions.
func (k *KeyVerifier) withPermissions(ctx context.Context, query rbac.PermissionQuery) error {
	_, span := tracing.Start(ctx, "verify.withPermissions")
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
	names := []string{}
	ratelimitRequests := []ratelimit.RatelimitRequest{}
	for name, config := range ratelimitsToCheck {
		names = append(names, name)
		ratelimitRequests = append(ratelimitRequests, ratelimit.RatelimitRequest{
			Name:       config.Name,
			Identifier: config.Identifier, // Use the pre-determined identifier
			Limit:      config.Limit,
			Duration:   config.Duration,
			Cost:       config.Cost,
			Time:       time.Time{}, // intentionally zero, so the ratelimiter will use its own clock
		})
	}

	// Use different rate limiting paths based on number of limits
	var resp []ratelimit.RatelimitResponse
	var err error

	if len(ratelimitRequests) == 1 {
		// Single rate limit - use fast path
		singleResp, singleErr := k.rateLimiter.Ratelimit(ctx, ratelimitRequests[0])
		resp = []ratelimit.RatelimitResponse{singleResp}
		err = singleErr
	} else {
		// Multiple rate limits - use atomic all-or-nothing path
		resp, err = k.rateLimiter.RatelimitMany(ctx, ratelimitRequests)
	}

	if err != nil {
		k.logger.Error("Failed to ratelimit",
			"key_id", k.Key.ID,
			"error", err.Error(),
		)

		// We will just allow the request to proceed, but log the error
		return nil
	}

	for i := range resp {
		// Write response back to config to be passed to the client
		config := ratelimitsToCheck[names[i]]
		config.Response = &resp[i]
		ratelimitsToCheck[names[i]] = config

		if !resp[i].Success {
			k.setInvalid(StatusRateLimited, fmt.Sprintf("key exceeded rate limit %s", names[i]))
		}
	}

	// Store the final results
	k.RatelimitResults = ratelimitsToCheck

	return nil
}
