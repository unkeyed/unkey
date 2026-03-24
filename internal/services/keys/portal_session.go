package keys

import (
	"context"
	"encoding/json"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
)

// GetPortalSession validates a portal session token and returns session info
// for scoping existing handlers by workspace and external user identity.
func (s *service) GetPortalSession(ctx context.Context, sess *zen.Session, token string) (*PortalSessionInfo, error) {
	ctx, span := tracing.Start(ctx, "keys.GetPortalSession")
	defer span.End()

	if token == "" {
		return nil, fault.New("empty session token",
			fault.Code(codes.Auth.Authentication.Missing.URN()),
			fault.Internal("portal session token is empty"),
			fault.Public("A valid portal session token is required."),
		)
	}

	row, err := db.Query.FindValidPortalSession(ctx, s.db.RO(), token)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.New("invalid or expired portal session",
				fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
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

	var permissions []string
	if row.Permissions != nil {
		if err := json.Unmarshal(row.Permissions, &permissions); err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to unmarshal portal session permissions"),
				fault.Public("An internal error occurred."),
			)
		}
	}

	var metadata map[string]any
	if row.Metadata != nil {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to unmarshal portal session metadata"),
				fault.Public("An internal error occurred."),
			)
		}
	}

	sess.WorkspaceID = row.WorkspaceID

	return &PortalSessionInfo{
		WorkspaceID:    row.WorkspaceID,
		ExternalID:     row.ExternalID,
		PortalConfigID: row.PortalConfigID,
		Permissions:    permissions,
		Metadata:       metadata,
		Preview:        row.Preview,
	}, nil
}
