// Package portaldomain holds what the v2/portal.*Domain routes share: resolving
// and authorizing the portal a request names.
//
// The control plane trusts the portal id it is handed, so every route must go
// through [ResolvePortal] before it calls ctrl.
package portaldomain

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// Access is the portal permission a route requires.
type Access int

const (
	// Read is required to get or list a portal's domains.
	Read Access = iota
	// Update is required to create, verify, or delete a portal's domain.
	Update
)

// portalNotFoundMessage matches the portal routes, so a denial here reads the
// same as one from portal.getPortal or portal.updatePortal.
const portalNotFoundMessage = "Portal not found."

// ResolvePortal finds the portal ref names in the caller's workspace and checks
// the caller holds access on it. A denial and an absent portal return the same
// fault, so the response cannot confirm a portal exists.
func ResolvePortal(
	ctx context.Context,
	database db.Database,
	p *principal.Principal,
	ref string,
	access Access,
) (db.Portal, error) {
	found, err := db.Query.FindPortalByIdOrSlug(ctx, database.RO(), db.FindPortalByIdOrSlugParams{
		Portal:      ref,
		WorkspaceID: p.AuthorizedWorkspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return db.Portal{}, fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal("no portal matched the request in this workspace"),
				fault.Public(portalNotFoundMessage),
			)
		}
		return db.Portal{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("We're unable to read the portal."),
		)
	}

	err = p.Authorize(rbac.Or(authorizationArms(p.AuthorizedWorkspaceID, found, access)...))
	if err != nil {
		// A fresh chain, not a wrap: wrapping would append the rendered RBAC
		// query, which names the resolved portal id, to the public message.
		return db.Portal{}, fault.New("portal not found",
			fault.Code(codes.Data.Portal.NotFound.URN()),
			fault.Internal(fmt.Sprintf("portal domain access denied for portal %s: %s", found.ID, fault.InternalMessage(err))),
			fault.Public(portalNotFoundMessage),
		)
	}

	return found, nil
}

// authorizationArms mirrors the portal routes' tuple and URN arms. Read also
// accepts update, because a key that can create a domain has to be able to poll
// it for its verification status.
func authorizationArms(workspaceID string, found db.Portal, access Access) []rbac.PermissionQuery {
	arms := func(tupleAction rbac.ActionType, urnAction permissions.Action) []rbac.PermissionQuery {
		return []rbac.PermissionQuery{
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   "*",
				Action:       tupleAction,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   found.ID,
				Action:       tupleAction,
			}),
			rbac.U(
				urn.New().Workspace(workspaceID).Project(found.ProjectID).Portal(found.ID),
				urnAction,
			),
		}
	}

	update := arms(rbac.UpdatePortal, permissions.Write)
	if access == Update {
		return update
	}
	return append(arms(rbac.ReadPortal, permissions.Read), update...)
}
