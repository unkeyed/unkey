// Package portaldomain holds what the v2/portal.*Domain routes share: looking
// up the portal a request names and the response a missing or denied portal
// gets.
//
// The control plane trusts the portal id it is handed, so every route must find
// the portal with [FindPortal] and authorize it before it calls ctrl.
package portaldomain

import (
	"context"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// portalNotFoundMessage matches the portal routes, so a denial here reads the
// same as one from portal.getPortal or portal.updatePortal.
const portalNotFoundMessage = "Portal not found."

// FindPortal finds the portal ref names, by id or slug, in workspaceID.
func FindPortal(ctx context.Context, database db.Database, workspaceID, ref string) (db.Portal, error) {
	found, err := db.Query.FindPortalByIdOrSlug(ctx, database.RO(), db.FindPortalByIdOrSlugParams{
		Portal:      ref,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return db.Portal{}, PortalNotFound("no portal matched the request in this workspace")
		}
		return db.Portal{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("We're unable to read the portal."),
		)
	}
	return found, nil
}

// PortalNotFound is the fault for an absent portal and for a denied one alike,
// so the response cannot confirm a portal exists. It starts a fresh chain:
// wrapping an RBAC denial would append the rendered query, which names the
// portal id, to the public message.
func PortalNotFound(detail string) error {
	return fault.New("portal not found",
		fault.Code(codes.Data.Portal.NotFound.URN()),
		fault.Internal(detail),
		fault.Public(portalNotFoundMessage),
	)
}
