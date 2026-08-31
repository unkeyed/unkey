package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalDeletePortalRequestBody
	Response = openapi.V2PortalDeletePortalResponseBody
)

// notFoundMessage is the single public message every unresolved delete returns.
//
// A denial, a portal in another workspace, an unknown id or slug, and a portal
// that was already deleted all share it, so no response body can be used to tell
// them apart.
const notFoundMessage = "Portal not found."

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.deletePortal"
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

	// The resolve, the delete, the session revocation, and the audit entry have to
	// be one atomic unit, so the closure is never replayed.
	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		found, err := db.Query.FindPortalByIdOrSlug(ctx, tx, db.FindPortalByIdOrSlugParams{
			Portal:      req.Portal,
			WorkspaceID: principal.WorkspaceID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("portal not found",
					fault.Code(codes.Data.Portal.NotFound.URN()),
					fault.Internal("no portal matched the request in this workspace"),
					fault.Public(notFoundMessage),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up portal"),
				fault.Public("We're unable to delete the portal."),
			)
		}

		// Resolved first, then authorized, so the query can name the concrete id a
		// scoped grant would carry. Safe because the resolve is workspace-scoped --
		// a foreign portal is already absent above -- Authorize is an in-memory
		// check over already-loaded permissions, and nothing has been written yet
		// The wildcard arm is spelled out separately because a stored `*`
		// matches literally and does not expand.
		//
		// The URN arms are what let the dashboard reach this route: its proxy mints
		// a token whose admin grant is a URN, so a legacy-only check would deny the
		// only operator surface there is.
		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   "*",
				Action:       rbac.DeletePortal,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   found.ID,
				Action:       rbac.DeletePortal,
			}),
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Portal("*"),
				permissions.DeletePortal{},
			),
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Portal(found.ID),
				permissions.DeletePortal{},
			),
		))
		if err != nil {
			// A fresh chain, not a wrap: UserFacingMessage concatenates every public
			// message in the chain, so wrapping would append the rendered RBAC query
			// -- which names the resolved portal id -- to the response. A caller may
			// have addressed the portal by slug and never seen that id. The internal
			// message carries the denial across so logs still distinguish it from a
			// genuinely absent portal.
			return fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal(fmt.Sprintf("delete denied for portal %s: %s", found.ID, fault.InternalMessage(err))),
				fault.Public(notFoundMessage),
			)
		}

		// Scoped on (id, workspace_id) so one workspace can never delete another's
		// row even if the resolve above were ever widened. There is no branding
		// cleanup to do: `logo_url` and `primary_color` are columns on this row, not
		// a side table, so the row going away takes the branding with it. Do not add
		// a second delete here.
		affected, err := db.Query.DeletePortal(ctx, tx, db.DeletePortalParams{
			ID:          found.ID,
			WorkspaceID: principal.WorkspaceID,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to delete portal"),
				fault.Public("We're unable to delete the portal."),
			)
		}

		// Nothing matched, so a concurrent delete already removed the row between
		// the resolve and here -- the resolve takes no row lock. Reported as
		// not-found rather than as a second success, so two racing deletes do not
		// both claim to have done it and write two audit entries for one deletion.
		if affected == 0 {
			return fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal(fmt.Sprintf("delete matched no rows for portal %s; concurrently deleted", found.ID)),
				fault.Public(notFoundMessage),
			)
		}

		// Unconditional, unlike the update path where revocation is tied to the
		// mapping changing: a deleted portal has no state left for a session to be
		// consistent with. The session resolver reads `portal_sessions` alone and
		// never `portals`, so without this every live end user would keep
		// authenticating against the deleted portal's frozen scope until the access
		// token expired. Scoped to this portal, so sessions belonging
		// to another portal in the same workspace are untouched, and it does not
		// cascade to the app or keyspace the portal served.
		revoked, err := db.Query.RevokePortalSessionsByPortal(ctx, tx, db.RevokePortalSessionsByPortalParams{
			RevokedAt:   sql.NullInt64{Valid: true, Int64: now},
			PortalID:    found.ID,
			WorkspaceID: principal.WorkspaceID,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to revoke portal sessions"),
				fault.Public("We're unable to delete the portal."),
			)
		}

		mappingType, mappingID := portal.DescribeMapping(found)

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.PortalDeleteEvent,
				Display:       fmt.Sprintf("Deleted portal %s", found.ID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:   found.ID,
						Type: auditlog.PortalResourceType,
						// The whole deleted state, because the row is gone: an incident
						// reviewer cannot recover which app or keyspace this portal served,
						// nor how many end users lost access, from anywhere else.
						Meta: map[string]any{
							"slug":            found.Slug,
							"mappingType":     mappingType,
							"mappingId":       mappingID,
							"enabled":         found.Enabled,
							"sessionsRevoked": revoked,
						},
						Name:        found.Slug,
						DisplayName: found.Slug,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
