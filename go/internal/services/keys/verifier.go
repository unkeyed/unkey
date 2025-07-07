package keys

import (
	"context"
	"database/sql"
	"strings"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"golang.org/x/exp/slices"
)

type KeyVerifier struct {
	Key db.FindKeyForVerificationRow // The actual key data embedded

	rateLimiter  ratelimit.Service
	usageLimiter usagelimiter.Service
	rBAC         *rbac.RBAC

	Valid  bool
	Status openapi.KeysVerifyKeyResponseDataCode

	error error
}

func (k *KeyVerifier) WithCredits(ctx context.Context, cost int32) *KeyVerifier {
	if !k.Valid || k.error != nil {
		return k
	}

	usage, err := k.usageLimiter.Limit(ctx, usagelimiter.UsageRequest{
		KeyId: k.Key.ID,
		Cost:  cost,
	})
	if err != nil {
		k.error = err
		return nil
	}

	k.Key.RemainingRequests = sql.NullInt32{Int32: usage.Remaining, Valid: true}
	if !usage.Valid {
		k.Valid = false
		k.Status = openapi.USAGEEXCEEDED
	}

	return k
}

func (k *KeyVerifier) WithIPWhitelist(ctx context.Context, ip string) *KeyVerifier {
	if !k.Valid || k.error != nil {
		return k
	}

	if !k.Key.IpWhitelist.Valid {
		return k
	}

	allowedIPs := strings.Split(k.Key.IpWhitelist.String, ",")
	if !slices.Contains(allowedIPs, ip) {
		k.Valid = false
		k.Status = openapi.FORBIDDEN
	}

	return k
}

func (k *KeyVerifier) WithPermissions(ctx context.Context, query rbac.PermissionQuery) *KeyVerifier {
	if !k.Valid || k.error != nil {
		return k
	}

	allowed, err := k.rBAC.EvaluatePermissions(query, k.Key.Perms.([]string))
	if err != nil {
		k.error = err
		return k
	}

	if !allowed.Valid {
		k.Valid = false
		k.Status = openapi.FORBIDDEN
	}

	return k
}

func (k *KeyVerifier) WithRateLimits(ctx context.Context, specifiedLimits []string) *KeyVerifier {
	if !k.Valid || k.error != nil {
		return k
	}

	return k
}

type Ratelimit struct {
}

func (k *KeyVerifier) Result() (*KeyVerifier, error) {
	return k, k.error
}

func (k *KeyVerifier) GetRoles() []string {
	return k.Key.Roles.([]string)
}

func (k *KeyVerifier) GetPermissions() []string {
	return k.Key.Perms.([]string)
}

func (k *KeyVerifier) GetRatelimits() []db.KeyFindForVerificationRatelimit {
	return k.Key.Ratelimits.([]db.KeyFindForVerificationRatelimit)
}
