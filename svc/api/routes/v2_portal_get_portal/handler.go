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

	target, err := parseTarget(req)
	if err != nil {
		return err
	}

	found, err := h.resolve(ctx, principal.AuthorizedWorkspaceID, target)
	if err != nil {
		return err
	}

	// Resolved first, then authorized, so the query can name the concrete ID a
	// scoped grant would carry. Safe because the resolve is workspace-scoped -- a
	// foreign portal is already absent above -- and Authorize is an in-memory
	// check over already-loaded permissions, so it adds no query and no timing
	// signature. The wildcard arm is spelled out separately because a stored `*`
	// matches literally and does not expand.
	//
	// Portals are not in the canonical URN catalog, so scoped access uses legacy
	// tuples. The exact admin permission lets the dashboard use this route. The
	// JWT admin role produces it.
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
		rbac.S(fmt.Sprintf("unkey:v1:%s:**#*", principal.AuthorizedWorkspaceID)),
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

// target is the one address a request carries: either the portal itself, or the
// single resource it serves.
//
// The wire shape has three optional fields, so "two of them" and "none of them"
// are both representable. This type is not: parseTarget is the only way to build
// one, so resolve cannot be handed an ambiguous request.
type target struct {
	portal  string
	mapping portal.Mapping
}

// parseTarget enforces exactly one of `portal`, `keyspaceId`, or `appId`.
//
// The dashboard holds an app or keyspace id and no portal id, so all three
// address forms have to exist. Accepting two at once would need a precedence
// rule, and either choice would silently ignore half of what the caller asked
// for. The `oneOf` in the spec rejects the same combinations, but it cannot
// express that a whitespace-only value is not a value.
func parseTarget(req Request) (target, error) {
	ambiguous := func(detail string) (target, error) {
		return target{portal: "", mapping: portal.Mapping{Type: "", ID: ""}},
			fault.New("ambiguous portal target",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(detail),
				fault.Public("Provide exactly one of `portal`, `keyspaceId`, or `appId`."),
			)
	}

	ref := ""
	if req.Portal != nil {
		ref = strings.TrimSpace(*req.Portal)
	}
	hasKeyspace := req.KeyspaceId != nil && strings.TrimSpace(string(*req.KeyspaceId)) != ""
	hasApp := req.AppId != nil && strings.TrimSpace(string(*req.AppId)) != ""

	named := 0
	for _, present := range []bool{ref != "", hasKeyspace, hasApp} {
		if present {
			named++
		}
	}
	if named != 1 {
		return ambiguous(fmt.Sprintf("portal=%t keyspaceId=%t appId=%t", ref != "", hasKeyspace, hasApp))
	}

	if ref != "" {
		return target{portal: ref, mapping: portal.Mapping{Type: "", ID: ""}}, nil
	}

	// Exactly one of the two ids is set, so this cannot report both-or-neither.
	mapping, err := portal.MappingFrom(
		(*string)(req.KeyspaceId),
		(*string)(req.AppId),
	)
	if err != nil {
		return target{portal: "", mapping: portal.Mapping{Type: "", ID: ""}}, err
	}
	return target{portal: "", mapping: mapping}, nil
}

// resolve finds the one portal the target addresses, scoped to the workspace.
//
// Read on the RO connection: this is the only statement, so there is no
// read-after-write to keep on the primary.
func (h *Handler) resolve(ctx context.Context, workspaceID string, t target) (db.Portal, error) {
	var (
		found db.Portal
		err   error
	)

	switch {
	case t.portal != "":
		found, err = db.Query.FindPortalByIdOrSlug(ctx, h.DB.RO(), db.FindPortalByIdOrSlugParams{
			Portal:      t.portal,
			WorkspaceID: workspaceID,
		})
	default:
		switch t.mapping.Type {
		case portal.MappingTypeApp:
			found, err = db.Query.FindPortalByApp(ctx, h.DB.RO(), db.FindPortalByAppParams{
				AppID:       sql.NullString{String: t.mapping.ID, Valid: true},
				WorkspaceID: workspaceID,
			})
		case portal.MappingTypeKeyspace:
			found, err = db.Query.FindPortalByKeyspace(ctx, h.DB.RO(), db.FindPortalByKeyspaceParams{
				KeyAuthID:   sql.NullString{String: t.mapping.ID, Valid: true},
				WorkspaceID: workspaceID,
			})
		default:
			return found, portal.ErrUnknownMappingType(t.mapping.Type)
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
