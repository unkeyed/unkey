package keys

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
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
	error   error

	rateLimiter  ratelimit.Service
	usageLimiter usagelimiter.Service
	rBAC         *rbac.RBAC
	clickhouse   clickhouse.ClickHouse
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

func (k *KeyVerifier) WithIPWhitelist() *KeyVerifier {
	if !k.Valid || k.error != nil {
		return k
	}

	if !k.Key.IpWhitelist.Valid {
		return k
	}

	allowedIPs := strings.Split(k.Key.IpWhitelist.String, ",")
	if !slices.Contains(allowedIPs, k.session.Location()) {
		k.Valid = false
		k.Status = openapi.FORBIDDEN
	}

	return k
}

func (k *KeyVerifier) WithPermissions(ctx context.Context, query rbac.PermissionQuery) *KeyVerifier {
	if !k.Valid || k.error != nil {
		return k
	}

	allowed, err := k.rBAC.EvaluatePermissions(query, k.Permissions)
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

func (k *KeyVerifier) Result() (*KeyVerifier, error) {
	k.clickhouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
		RequestID:   k.session.RequestID(),
		WorkspaceID: k.session.AuthorizedWorkspaceID(),
		// ForWorkspaceID: s.AuthorizedWorkspaceID, // We need this at some point to actually assign the logs to the "real" workspaceID
		Time:       time.Now().UnixMilli(),
		Region:     "",
		Outcome:    "",
		KeySpaceID: k.Key.KeyAuthID,
		KeyID:      k.Key.ID,
		IdentityID: k.Key.IdentityID.String,
		Tags:       []string{},
	})

	return k, k.error
}
