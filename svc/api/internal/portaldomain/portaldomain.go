// Package portaldomain holds what the v2/portal.*Domain routes share: resolving
// and authorizing the portal a request names, loading one of its domains, and
// rendering a domain row for the API.
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
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	"github.com/unkeyed/unkey/svc/api/openapi"
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

const domainNotFoundMessage = "The requested domain does not exist."

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

// FindDomain loads one domain attached to portalID. A domain on another portal
// or in another workspace is not found.
func FindDomain(
	ctx context.Context,
	database db.Database,
	workspaceID, portalID, domainID string,
) (db.FindPortalDomainByIdRow, error) {
	row, err := db.Query.FindPortalDomainById(ctx, database.RO(), db.FindPortalDomainByIdParams{
		ID:          domainID,
		WorkspaceID: workspaceID,
		PortalID:    portalID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return row, DomainNotFound(fmt.Sprintf("no domain %s on portal %s", domainID, portalID))
		}
		return row, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal domain"),
			fault.Public("Failed to retrieve domain."),
		)
	}
	return row, nil
}

// DomainNotFound is the fault for a domain id that does not resolve on the
// requested portal.
func DomainNotFound(detail string) error {
	return fault.New("portal domain not found",
		fault.Code(codes.Data.Domain.NotFound.URN()),
		fault.Internal(detail),
		fault.Public(domainNotFoundMessage),
	)
}

// ToResponse renders a stored portal domain.
func ToResponse(row db.FindPortalDomainByIdRow) openapi.PortalDomain {
	data := openapi.PortalDomain{
		Id:                row.ID,
		PortalId:          row.PortalID,
		Domain:            row.Domain,
		Status:            status(row.VerificationStatus),
		VerificationError: nil,
		DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
			Domain:            row.Domain,
			TargetCname:       row.TargetCname,
			VerificationToken: row.VerificationToken,
			RoutingVerified:   row.CnameVerified,
			OwnershipVerified: row.OwnershipVerified,
		}),
		CreatedAt: row.CreatedAt,
		UpdatedAt: nil,
	}
	if row.VerificationError.Valid && row.VerificationError.String != "" {
		data.VerificationError = new(row.VerificationError.String)
	}
	if row.UpdatedAt.Valid {
		data.UpdatedAt = new(row.UpdatedAt.Int64)
	}
	return data
}

// status reports an unrecognised value as pending rather than inventing a
// terminal state a caller would act on.
func status(s db.PortalDomainsVerificationStatus) openapi.PortalDomainStatus {
	switch s {
	case db.PortalDomainsVerificationStatusVerifying:
		return openapi.PortalDomainStatusVerifying
	case db.PortalDomainsVerificationStatusVerified:
		return openapi.PortalDomainStatusVerified
	case db.PortalDomainsVerificationStatusFailed:
		return openapi.PortalDomainStatusFailed
	case db.PortalDomainsVerificationStatusPending:
		return openapi.PortalDomainStatusPending
	default:
		return openapi.PortalDomainStatusPending
	}
}
