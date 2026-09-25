package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalRevokeSessionRequestBody
	Response = openapi.V2PortalRevokeSessionResponseBody
)

// notFoundMessage is the single public message for an unknown portal and for a
// caller who may not revoke its sessions, so neither can be told apart.
const notFoundMessage = "Portal not found."

type Handler struct {
	DB           db.Database
	Auditlogs    auditlogs.AuditLogService
	Clock        clock.Clock
	SessionCache cache.Cache[string, db.PortalSession]
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.revokeSession"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	now := h.Clock.Now().UnixMilli()
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	revoked, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) ([]db.PortalSession, error) {
		found, err := db.Query.FindPortalByIdOrSlug(ctx, tx, db.FindPortalByIdOrSlugParams{
			Portal:      req.Portal,
			WorkspaceID: principal.AuthorizedWorkspaceID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return nil, fault.New("portal not found",
					fault.Code(codes.Data.Portal.NotFound.URN()),
					fault.Internal("no portal matched the request in this workspace"),
					fault.Public(notFoundMessage),
				)
			}
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up portal"),
				fault.Public("We're unable to revoke the portal sessions."),
			)
		}

		// The same grant that mints a session for this portal revokes one. Unlike
		// minting there is no root-key-only guard: revoking removes access rather
		// than acting as the end user, so a dashboard admin may do it too.
		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   "*",
				Action:       rbac.CreatePortalSession,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   found.ID,
				Action:       rbac.CreatePortalSession,
			}),
			rbac.U(
				urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(found.ProjectID).Portal(found.ID).Session("*"),
				permissions.Write,
			),
		))
		if err != nil {
			return nil, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Portal.NotFound.URN(), notFoundMessage)
		}

		count, err := db.Query.RevokePortalSessionsByExternalID(ctx, tx, db.RevokePortalSessionsByExternalIDParams{
			RevokedAt:                sql.NullInt64{Valid: true, Int64: now},
			WorkspaceID:              principal.AuthorizedWorkspaceID,
			PortalID:                 found.ID,
			ExternalID:               req.ExternalId,
			AccessTokenExpiresAfter:  sql.NullInt64{Valid: true, Int64: now},
			ExchangeCodeExpiresAfter: now,
		})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to revoke portal sessions"),
				fault.Public("We're unable to revoke the portal sessions."),
			)
		}
		if count == 0 {
			return nil, nil
		}

		sessions, err := db.Query.FindPortalSessionsRevokedAtByExternalID(ctx, tx, db.FindPortalSessionsRevokedAtByExternalIDParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			PortalID:    found.ID,
			ExternalID:  req.ExternalId,
			RevokedAt:   sql.NullInt64{Valid: true, Int64: now},
		})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to read revoked portal sessions"),
				fault.Public("We're unable to revoke the portal sessions."),
			)
		}

		sessionIDs := make([]string, 0, len(sessions))
		for _, session := range sessions {
			sessionIDs = append(sessionIDs, session.ID)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.PortalSessionRevokeEvent,
				Display:       fmt.Sprintf("Revoked portal sessions for %s", req.ExternalId),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          found.ID,
						Type:        auditlog.PortalResourceType,
						Name:        found.Slug,
						DisplayName: found.Slug,
						Meta: map[string]any{
							"externalId":      req.ExternalId,
							"sessionIds":      sessionIDs,
							"sessionsRevoked": len(sessions),
						},
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return sessions, nil
	})
	if err != nil {
		return err
	}

	// Written through rather than removed: a removed entry refills from the
	// read replica, which may not have the revocation yet.
	cached := make(map[string]db.PortalSession, len(revoked))
	for _, session := range revoked {
		if session.AccessTokenHash.Valid {
			cached[session.AccessTokenHash.String] = session
		}
	}
	h.SessionCache.SetMany(ctx, cached)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2PortalRevokeSessionResponseData{
			SessionsRevoked: int64(len(revoked)),
		},
	})
}
