package portal

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

// GetSession validates a portal session token and returns session info
// for scoping existing handlers by workspace and external user identity.
func (s *service) GetSession(ctx context.Context, token string) (*SessionInfo, error) {
	ctx, span := tracing.Start(ctx, "portal.GetSession")
	defer span.End()

	if token == "" {
		return nil, fault.New("empty session token",
			fault.Code(codes.Portal.Session.TokenMissing.URN()),
			fault.Internal("portal session token is empty"),
			fault.Public("A valid portal session token is required."),
		)
	}

	row, hit, err := s.sessionCache.SWR(ctx, token, func(ctx context.Context) (db.PortalSession, error) {
		return db.Query.FindValidPortalSession(ctx, s.db.RO(), db.FindValidPortalSessionParams{
			ID:  token,
			Now: time.Now().UnixMilli(),
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

	permissions, err := db.UnmarshalNullableJSONTo[[]string](row.Permissions)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal portal session permissions"),
			fault.Public("An internal error occurred."),
		)
	}

	return &SessionInfo{
		WorkspaceID:    row.WorkspaceID,
		ExternalID:     row.ExternalID,
		PortalConfigID: row.PortalConfigID,
		Permissions:    permissions,
		Preview:        row.Preview,
	}, nil
}
