package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
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

// revokeBatchSize caps how many sessions one transaction revokes, so an end user
// with a very large number of sessions doesn't hold one huge lock.
const revokeBatchSize = 1000

// notFoundMessage is shared by an unknown portal and a denied caller, so the two
// look the same.
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

	found, err := h.authorizedPortal(ctx, principal, req.Portal)
	if err != nil {
		return err
	}

	// Bounded by pk rather than created_at: created_at comes from whichever
	// instance minted the session, and its clock can run ahead of this one.
	maxPk, err := db.Query.MaxPortalSessionPkByExternalID(ctx, h.DB.RW(), db.MaxPortalSessionPkByExternalIDParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		PortalID:    found.ID,
		ExternalID:  req.ExternalId,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("unable to read portal session bound for portal %s", found.ID)),
			fault.Public("We're unable to revoke the portal sessions."),
		)
	}

	var total int64
	for done := maxPk == 0; !done; {
		revoked, err := h.revokeBatch(ctx, s, principal, found, req.ExternalId, uint64(maxPk), now)
		if err != nil {
			return err
		}
		total += int64(len(revoked))

		// Written through, not evicted: a refill could read a replica that doesn't
		// have the revocation yet.
		for _, session := range revoked {
			if session.AccessTokenHash.Valid {
				h.SessionCache.Set(ctx, session.AccessTokenHash.String, session)
			}
		}

		done = len(revoked) < revokeBatchSize
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2PortalRevokeSessionResponseData{
			SessionsRevoked: total,
		},
	})
}

// authorizedPortal resolves the portal in the caller's workspace and checks the
// caller may revoke its sessions. Both misses return the same 404.
func (h *Handler) authorizedPortal(ctx context.Context, principal *authprincipal.Principal, target string) (db.Portal, error) {
	found, err := db.Query.FindPortalByIdOrSlug(ctx, h.DB.RW(), db.FindPortalByIdOrSlugParams{
		Portal:      target,
		WorkspaceID: principal.AuthorizedWorkspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return db.Portal{}, fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal("no portal matched the request in this workspace"),
				fault.Public(notFoundMessage),
			)
		}
		return db.Portal{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("We're unable to revoke the portal sessions."),
		)
	}

	// Same grant as minting, but no root-key-only guard: revoking only removes
	// access, so a dashboard admin may do it too.
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
		return db.Portal{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Portal.NotFound.URN(), notFoundMessage)
	}

	return found, nil
}

// revokeBatch revokes up to revokeBatchSize of the end user's live sessions at
// or below maxPk in one transaction, writes their audit entry, and returns them
// with revoked_at set. It returns no rows once nothing is left to revoke.
func (h *Handler) revokeBatch(
	ctx context.Context,
	s *zen.Session,
	principal *authprincipal.Principal,
	found db.Portal,
	externalID string,
	maxPk uint64,
	now int64,
) ([]db.PortalSession, error) {
	return db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) ([]db.PortalSession, error) {
		sessions, err := db.Query.LockLivePortalSessionsByExternalID(ctx, tx, db.LockLivePortalSessionsByExternalIDParams{
			WorkspaceID:              principal.AuthorizedWorkspaceID,
			PortalID:                 found.ID,
			ExternalID:               externalID,
			MaxPk:                    maxPk,
			AccessTokenExpiresAfter:  sql.NullInt64{Valid: true, Int64: now},
			ExchangeCodeExpiresAfter: now,
			Limit:                    revokeBatchSize,
		})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("unable to lock portal sessions for portal %s", found.ID)),
				fault.Public("We're unable to revoke the portal sessions."),
			)
		}
		if len(sessions) == 0 {
			return nil, nil
		}

		sessionIDs := make([]string, 0, len(sessions))
		for i := range sessions {
			sessionIDs = append(sessionIDs, sessions[i].ID)
			sessions[i].RevokedAt = sql.NullInt64{Valid: true, Int64: now}
		}

		count, err := db.Query.RevokePortalSessionsByIDs(ctx, tx, db.RevokePortalSessionsByIDsParams{
			RevokedAt:   sql.NullInt64{Valid: true, Int64: now},
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Ids:         sessionIDs,
		})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("unable to revoke portal sessions for portal %s", found.ID)),
				fault.Public("We're unable to revoke the portal sessions."),
			)
		}
		if err = assert.Equal(count, int64(len(sessions)), "revoke must affect exactly the locked sessions"); err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal(fmt.Sprintf("revoked %d portal sessions for portal %s but locked %d", count, found.ID, len(sessions))),
				fault.Public("We're unable to revoke the portal sessions."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.PortalSessionRevokeEvent,
				Display:       fmt.Sprintf("Revoked portal sessions for %s", externalID),
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
							"externalId":      externalID,
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
}
