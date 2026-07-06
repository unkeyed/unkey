// Package identityattr resolves caller-supplied ratelimit identifiers to
// end-user identities so standalone ratelimit events can be attributed for
// per-identity billing.
package identityattr

import (
	"context"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

// Resolve maps a caller-supplied ratelimit identifier to an end-user identity
// by matching it against the workspace's identity external_ids — the
// documented convention is that customers use their end-user's id as the
// ratelimit identifier. Identifiers that match no identity resolve to empty
// strings and the event stays unattributed.
//
// Attribution is best-effort: cache misses fall through to a single indexed
// MySQL lookup (negative results are cached as null entries), and any lookup
// failure returns empty strings — it must never fail the ratelimit request.
func Resolve(ctx context.Context, c cache.Cache[cache.ScopedKey, db.Identity], database db.Database, workspaceID, identifier string) (identityID, externalID string) {
	if identifier == "" {
		return "", ""
	}

	identity, hit, err := c.SWR(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: identifier}, func(ctx context.Context) (db.Identity, error) {
		return db.WithRetryContext(ctx, func() (db.Identity, error) {
			return db.Query.FindIdentityByExternalID(ctx, database.RO(), db.FindIdentityByExternalIDParams{
				WorkspaceID: workspaceID,
				ExternalID:  identifier,
				Deleted:     false,
			})
		})
	}, caches.DefaultFindFirstOp)
	if err != nil || hit == cache.Null {
		return "", ""
	}

	return identity.ID, identity.ExternalID
}
