// Package portaldomain holds what the v2/portal.*Domain routes share: looking
// up the portal a request names, loading one of its domains, and rendering a
// domain row for the API.
//
// The control plane trusts the portal id it is handed, so every route must find
// the portal with [FindPortal] and authorize it before it calls ctrl.
package portaldomain

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// PortalNotFoundMessage matches the portal routes, so a denial here reads the
// same as one from portal.getPortal or portal.updatePortal.
const PortalNotFoundMessage = "Portal not found."

const domainNotFoundMessage = "The requested domain does not exist."

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
		fault.Public(PortalNotFoundMessage),
	)
}

// Lookup is the portal a request names and the domain it asks for.
//
// DomainErr is held back rather than returned so the caller can authorize on
// Portal first: a caller denied the portal must not learn whether a domain id
// exists.
type Lookup struct {
	Portal    db.Portal
	Domain    db.FindPortalDomainByIdRow
	DomainErr error
}

// FindPortalAndDomain reads the portal portalRef names and the domain domainID
// concurrently. The returned error is the portal's; a domain that is missing or
// attached to another portal is reported in [Lookup.DomainErr].
func FindPortalAndDomain(
	ctx context.Context,
	database db.Database,
	workspaceID, portalRef, domainID string,
) (Lookup, error) {
	var lookup Lookup
	var portalErr error
	err := conc.All(ctx,
		func(ctx context.Context) error {
			lookup.Portal, portalErr = FindPortal(ctx, database, workspaceID, portalRef)
			return nil
		},
		func(ctx context.Context) error {
			lookup.Domain, lookup.DomainErr = findDomain(ctx, database, workspaceID, domainID)
			return nil
		},
	)
	if err != nil {
		return Lookup{}, err
	}
	if portalErr != nil {
		return Lookup{}, portalErr
	}
	if lookup.DomainErr == nil && lookup.Domain.PortalID != lookup.Portal.ID {
		lookup.DomainErr = DomainNotFound(fmt.Sprintf("domain %s is not on portal %s", domainID, lookup.Portal.ID))
	}
	return lookup, nil
}

// findDomain loads one domain in workspaceID.
func findDomain(ctx context.Context, database db.Database, workspaceID, domainID string) (db.FindPortalDomainByIdRow, error) {
	row, err := db.Query.FindPortalDomainById(ctx, database.RO(), db.FindPortalDomainByIdParams{
		ID:          domainID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return row, DomainNotFound(fmt.Sprintf("no domain %s in workspace %s", domainID, workspaceID))
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
		Status:            domain.PortalStatus(row.VerificationStatus),
		VerificationError: nil,
		DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
			Domain:            row.Domain,
			TargetCname:       row.TargetCname,
			VerificationToken: row.VerificationToken,
			RoutingVerified:   row.CnameVerified,
			OwnershipVerified: row.OwnershipVerified,
		}),
		DomainConnect: nil,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     nil,
	}
	if row.DomainConnectProvider.Valid && row.DomainConnectUrl.Valid {
		data.DomainConnect = &openapi.DomainConnect{
			Provider: row.DomainConnectProvider.String,
			Url:      row.DomainConnectUrl.String,
		}
	}
	if row.VerificationError.Valid && row.VerificationError.String != "" {
		data.VerificationError = new(row.VerificationError.String)
	}
	if row.UpdatedAt.Valid {
		data.UpdatedAt = new(row.UpdatedAt.Int64)
	}
	return data
}
