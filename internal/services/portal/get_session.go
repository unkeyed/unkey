package portal

import (
	"context"

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

	row, err := s.findSession(ctx, token)
	if err != nil {
		return nil, err
	}

	permissions, err := db.UnmarshalNullableJSONTo[[]string](row.Permissions)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal portal session permissions"),
			fault.Public("An internal error occurred."),
		)
	}

	metadata, err := db.UnmarshalNullableJSONTo[map[string]any](row.Metadata)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal portal session metadata"),
			fault.Public("An internal error occurred."),
		)
	}

	return &SessionInfo{
		WorkspaceID:    row.WorkspaceID,
		ExternalID:     row.ExternalID,
		PortalConfigID: row.PortalConfigID,
		Permissions:    permissions,
		Metadata:       metadata,
		Preview:        row.Preview,
	}, nil
}

// findSession looks up a portal session, using the SWR cache when available.
func (s *service) findSession(ctx context.Context, token string) (db.PortalSession, error) {
	if s.sessionCache != nil {
		row, hit, err := s.sessionCache.SWR(ctx, token, func(ctx context.Context) (db.PortalSession, error) {
			return db.Query.FindValidPortalSession(ctx, s.db.RO(), token)
		}, caches.DefaultFindFirstOp)
		if err != nil {
			return db.PortalSession{}, s.wrapLookupError(err)
		}
		if hit == cache.Null {
			return db.PortalSession{}, s.sessionNotFoundError("cached null")
		}
		return row, nil
	}

	row, err := db.Query.FindValidPortalSession(ctx, s.db.RO(), token)
	if err != nil {
		return db.PortalSession{}, s.wrapLookupError(err)
	}
	return row, nil
}

func (s *service) wrapLookupError(err error) error {
	if db.IsNotFound(err) {
		return s.sessionNotFoundError("not found or expired")
	}
	return fault.Wrap(err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("database error finding portal session"),
		fault.Public("Failed to validate portal session."),
	)
}

func (s *service) sessionNotFoundError(detail string) error {
	return fault.New("invalid or expired portal session",
		fault.Code(codes.Portal.Session.SessionNotFound.URN()),
		fault.Internal("portal session "+detail),
		fault.Public("The portal session is invalid or has expired."),
	)
}
