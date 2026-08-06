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
		return nil, fault.New("empty access token",
			fault.Code(codes.Portal.Session.TokenMissing.URN()),
			fault.Internal("portal access token is empty"),
			fault.Public("A valid portal access token is required."),
		)
	}

	accessTokenHash := hash.Sha256(accessToken)
	row, hit, err := s.sessionCache.SWR(ctx, accessTokenHash, func(ctx context.Context) (db.PortalSession, error) {
		return db.Query.FindValidPortalSession(ctx, s.db.RO(), db.FindValidPortalSessionParams{
			AccessTokenHash: sql.NullString{String: accessTokenHash, Valid: true},
			Now:             sql.NullInt64{Int64: s.clock.Now().UnixMilli(), Valid: true},
		})
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.New("invalid or expired portal session",
				fault.Code(codes.Portal.Session.SessionNotFound.URN()),
				fault.Internal("portal session not found or expired"),
				fault.Public("The portal session is invalid or has expired."),
			)
		}
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error finding portal session"),
			fault.Public("Failed to validate portal session."),
		)
	}
	if hit == cache.Null {
		return nil, fault.New("invalid or expired portal session",
			fault.Code(codes.Portal.Session.SessionNotFound.URN()),
			fault.Internal("portal session cached null"),
			fault.Public("The portal session is invalid or has expired."),
		)
	}
	if !row.AccessTokenExpiresAt.Valid || row.AccessTokenExpiresAt.Int64 <= s.clock.Now().UnixMilli() {
		return nil, fault.New("invalid or expired portal session",
			fault.Code(codes.Portal.Session.SessionNotFound.URN()),
			fault.Internal("portal session cached after expiry"),
			fault.Public("The portal session is invalid or has expired."),
		)
	}

	// The permissions column stores the simplified capability model as a JSON
	// object {keyspaceIds, permissions:[verbs]}, written by portal.createSession.
	// The resolver expands the verbs into RBAC via portalrbac.
	//
	// TODO: Re-evaluate this grant against the portal's current configuration
	// before authorization. Sessions snapshot keyspace IDs and permissions when
	// created, so configuration changes do not affect active sessions until they
	// expire. Intersect with the current configuration or add grant versioning
	// before configuration changes are expected to take effect immediately.
	grant, err := db.UnmarshalNullableJSONTo[storedGrant](row.Permissions)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal portal session grant"),
			fault.Public("An internal error occurred."),
		)
	}

	return &SessionInfo{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		ExternalID:  row.ExternalID,
		PortalID:    row.PortalID,
		KeyspaceIDs: grant.KeyspaceIDs,
		Permissions: grant.Permissions,
	}, nil
}

// storedGrant is the JSON object shape stored on the portal session's
// permissions column. It must stay in sync with the shape written by
// portal.createSession.
type storedGrant struct {
	KeyspaceIDs []string `json:"keyspaceIds"`
	Permissions []string `json:"permissions"`
}
