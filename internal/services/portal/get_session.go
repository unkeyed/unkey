package portal

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

// GetSession validates a portal access token and returns session info
// for scoping existing handlers by workspace and external user identity.
func (s *service) GetSession(ctx context.Context, accessToken string) (*SessionInfo, error) {
	ctx, span := tracing.Start(ctx, "portal.GetSession")
	defer span.End()

	if accessToken == "" {
		return nil, fault.New("empty portal access token",
			fault.Code(codes.Portal.Session.TokenMissing.URN()),
			fault.Internal("portal access token is empty"),
			fault.Public("A valid portal session token is required."),
		)
	}

	// The token is only ever stored and cached as its hash, so the plaintext
	// credential reaches neither MySQL nor the cache.
	accessTokenHash := hash.Sha256(accessToken)

	row, hit, err := s.sessionCache.SWR(ctx, accessTokenHash, func(ctx context.Context) (db.PortalSession, error) {
		return db.Query.FindPortalSessionByAccessTokenHash(ctx, s.db.RO(),
			sql.NullString{String: accessTokenHash, Valid: true})
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, invalidSessionError("portal session not found")
		}
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error finding portal session"),
			fault.Public("Failed to validate portal session."),
		)
	}
	if hit == cache.Null {
		return nil, invalidSessionError("portal session cached null")
	}

	// State is derived here rather than filtered in SQL: the row is cached, so a
	// query-time predicate would pin expiry and revocation to whatever was true
	// at fill time. Revocation therefore takes effect within the cache TTL.
	state := stateOf(row, s.clock.Now().UnixMilli())

	// Revocation outranks corruption. It is a deliberate, expected end state, so
	// a revoked row is rejected the same way whatever its timestamps look like;
	// reporting an internal error for it would page someone over a session that
	// was shut off on purpose.
	if state == StateRevoked {
		return nil, invalidSessionError("portal session is " + string(state))
	}

	// An access token hash without both of its timestamps is a corrupt row, not
	// a state the caller should be asked to interpret. Asserted before the
	// remaining states because stateOf fails closed on a missing expiry, which
	// would otherwise bury the fault as an ordinary expired session. The guard on
	// AccessTokenHash is load-bearing: a pending row legitimately carries neither
	// timestamp.
	if row.AccessTokenHash.Valid && (!row.AccessTokenCreatedAt.Valid || !row.AccessTokenExpiresAt.Valid) {
		return nil, fault.New("corrupt portal session row",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("access_token_hash is set but access_token_created_at or access_token_expires_at is NULL"),
			fault.Public("An internal error occurred."),
		)
	}

	if state != StateActive {
		return nil, invalidSessionError("portal session is " + string(state))
	}

	grant, err := db.UnmarshalNullableJSONTo[Grant](row.Scopes)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal portal session grant"),
			fault.Public("An internal error occurred."),
		)
	}

	return &SessionInfo{
		SessionID:   row.ID,
		WorkspaceID: row.WorkspaceID,
		ExternalID:  row.ExternalID,
		PortalID:    row.PortalID,
		Preview:     row.Preview,
		KeyspaceIDs: grant.KeyspaceIDs,
		Scopes:      grant.Scopes,
	}, nil
}

// invalidSessionError builds the single public failure every non-active session
// resolves to. The internal detail distinguishes the cases for operators; the
// public message deliberately does not, so a caller cannot probe which of
// "unknown", "expired", or "revoked" a token is.
func invalidSessionError(internal string) error {
	return fault.New("invalid or expired portal session",
		fault.Code(codes.Portal.Session.SessionNotFound.URN()),
		fault.Internal(internal),
		fault.Public("The portal session is invalid or has expired."),
	)
}
