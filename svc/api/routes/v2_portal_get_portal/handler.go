package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

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
	Request  = openapi.V2PortalGetPortalRequestBody
	Response = openapi.V2PortalGetPortalResponseBody
)

// notFoundMessage is the single public message every unresolved read returns.
//
// A denial, a portal in another workspace, an unknown id, and a mapping with no
// portal all share it, so no response body can be used to tell those four apart.
const notFoundMessage = "Portal not found."

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.getPortal"
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

	// The dashboard holds an app or keyspace id and no portal id, so both address
	// forms have to exist. Accepting both at once would need a precedence rule,
	// and either choice would silently ignore half of what the caller asked for.
	hasPortal := req.Portal != nil && strings.TrimSpace(*req.Portal) != ""
	hasMapping := req.Mapping != nil
	if hasPortal == hasMapping {
		return fault.New("ambiguous portal target",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("hasPortal=%t hasMapping=%t", hasPortal, hasMapping)),
			fault.Public("Provide exactly one of `portal` or `mapping`."),
		)
	}

	found, err := h.resolve(ctx, principal.WorkspaceID, req)
	if err != nil {
		return err
	}

	// Resolved first, then authorized, so the query can name the concrete id a
	// scoped grant would carry. Safe because the resolve is workspace-scoped -- a
	// foreign portal is already absent above -- and Authorize is an in-memory
	// check over already-loaded permissions, so it adds no query and no timing
	// signature. The wildcard arm is spelled out separately because a stored `*`
	// matches literally and does not expand.
	//
	// The URN arm is what lets the dashboard reach this route: its proxy mints a
	// token whose admin grant is a URN, so a legacy-only check would deny the only
	// operator surface there is.
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   "*",
			Action:       rbac.ReadPortal,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   found.ID,
			Action:       rbac.ReadPortal,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Portal("*"),
			permissions.ReadPortal{},
		),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Portal(found.ID),
			permissions.ReadPortal{},
		),
	))
	if err != nil {
		// A fresh chain, not a wrap: UserFacingMessage concatenates every public
		// message in the chain, so wrapping would append the rendered RBAC query --
		// which names the resolved portal id -- to the response. A caller may have
		// addressed the portal by slug or by mapping and never seen that id. The
		// internal message carries the denial across so logs still distinguish it
		// from a genuinely absent portal.
		return fault.New("portal not found",
			fault.Code(codes.Data.Portal.NotFound.URN()),
			fault.Internal(fmt.Sprintf("read denied for portal %s: %s", found.ID, fault.InternalMessage(err))),
			fault.Public(notFoundMessage),
		)
	}

	data, err := portal.ToResponse(found)
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}

// resolve finds the one portal the request addresses, scoped to the workspace.
//
// Read on the RO connection: this is the only statement, so there is no
// read-after-write to keep on the primary.
func (h *Handler) resolve(ctx context.Context, workspaceID string, req Request) (db.Portal, error) {
	var (
		found db.Portal
		err   error
	)

	// Mirrors the exactly-one check in Handle, including the trim: a
	// whitespace-only target is not a target, and taking this arm for one would
	// query for the empty string instead of using the mapping.
	switch {
	case req.Portal != nil && strings.TrimSpace(*req.Portal) != "":
		found, err = db.Query.FindPortalByIdOrSlug(ctx, h.DB.RO(), db.FindPortalByIdOrSlugParams{
			Portal:      strings.TrimSpace(*req.Portal),
			WorkspaceID: workspaceID,
		})
	default:
		switch req.Mapping.Type {
		case openapi.PortalMappingTypeApp:
			found, err = db.Query.FindPortalByApp(ctx, h.DB.RO(), db.FindPortalByAppParams{
				AppID:       sql.NullString{String: req.Mapping.Id, Valid: true},
				WorkspaceID: workspaceID,
			})
		case openapi.PortalMappingTypeKeyspace:
			found, err = db.Query.FindPortalByKeyspace(ctx, h.DB.RO(), db.FindPortalByKeyspaceParams{
				KeyAuthID:   sql.NullString{String: req.Mapping.Id, Valid: true},
				WorkspaceID: workspaceID,
			})
		default:
			return found, fault.New("unknown portal mapping type",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("unknown mapping type %q", req.Mapping.Type)),
				fault.Public(portal.ErrMsgInvalidMapping),
			)
		}
	}

	if err != nil {
		if db.IsNotFound(err) {
			// Deliberately identical to the denial above, and it does not say
			// whether the app or keyspace itself exists: the mapping arm is
			// workspace-scoped, so "no portal here" and "not your resource" have to
			// read the same or the response answers what the caller owns elsewhere.
			return found, fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal("no portal matched the request in this workspace"),
				fault.Public(notFoundMessage),
			)
		}
		return found, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("We're unable to read the portal."),
		)
	}

	return found, nil
}
