package principal

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth/ipallowlist"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

func RequireIPAllowlistEnabled(ctx context.Context, database db.DBTX, workspaceID string) error {
	enabled, err := db.Query.FindIPAllowlistEnabled(ctx, database, workspaceID)
	if err != nil && !db.IsNotFound(err) {
		return err
	}
	if !enabled {
		return fault.New("root key IP allowlist is not enabled", fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public("IP allowlists are not enabled for this workspace. Contact support to enable them."))
	}
	return nil
}

// ValidateRootKeyIPAllowlist bounds delegated access by the locked caller policy.
func ValidateRootKeyIPAllowlist(parent, next ipallowlist.Allowlist) error {
	if !next.IsSubsetOf(parent) {
		return fault.New("root key operation widens network access", fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public("The IP allowlist must stay within the calling root key's allowed networks."))
	}
	return nil
}
