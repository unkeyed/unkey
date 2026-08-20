// Package portal holds the rules and mapping the operator-facing portal routes
// share: what a portal may be mapped to, whether the caller owns that mapping,
// how branding input is validated, and how a stored row becomes the public
// shape.
//
// These live together because each one is an invariant rather than a
// convenience. A portal maps to exactly one app or keyspace and the database
// cannot enforce it (Vitess has no CHECK constraint), the app and keyspace
// unique keys span every workspace so an unvalidated mapping is a cross-tenant
// claim, and branding strings are rendered in end users' browsers with no
// Content-Security-Policy behind them. A second copy of any of these drifting
// out of step with the first is the failure this package exists to prevent.
package portal

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Public messages. A mapping that names a resource the caller does not own is
// deliberately reported as a plain not-found, identical to naming one that does
// not exist anywhere: a caller must not be able to use the difference to learn
// that another workspace holds it.
const (
	ErrMsgInvalidMapping  = "A portal must map to exactly one app or one keyspace."
	ErrMsgMappingNotFound = "The app or keyspace was not found."
	ErrMsgInvalidLogoURL  = "logoUrl must be an absolute https:// URL of at most 500 characters."
	ErrMsgInvalidColor    = "primaryColor must be a six-digit hex colour, for example #6366f1."
)

// LogoURLMaxLength matches the column width. Enforced here so an over-long value
// is a validation error rather than a truncation or a driver error surfacing as a
// 500.
const LogoURLMaxLength = 500

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// ColumnsFor turns a request's mapping into the two nullable columns that carry
// it. Exactly one is set.
//
// The request shape names the kind and the id together, so "both" and "neither"
// are unrepresentable rather than merely rejected. What remains reachable is an
// unknown kind, which a client could send past the generated enum.
func ColumnsFor(m openapi.PortalMapping) (appID sql.NullString, keyAuthID sql.NullString, err error) {
	id := strings.TrimSpace(m.Id)
	if id == "" {
		return sql.NullString{}, sql.NullString{}, fault.New("empty portal mapping id",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("mapping id was empty or whitespace"),
			fault.Public(ErrMsgInvalidMapping),
		)
	}

	switch m.Type {
	case openapi.PortalMappingTypeApp:
		return sql.NullString{String: id, Valid: true}, sql.NullString{}, nil
	case openapi.PortalMappingTypeKeyspace:
		return sql.NullString{}, sql.NullString{String: id, Valid: true}, nil
	default:
		return sql.NullString{}, sql.NullString{}, ErrUnknownMappingType(m.Type)
	}
}

// MappingOf reads the mapping back off a stored row.
//
// A row with both columns set, or neither, violates the invariant the
// application is solely responsible for. Rows written before these routes
// existed were never checked, so this refuses to guess which half to believe
// rather than serving an ambiguous portal.
func MappingOf(p db.Portal) (openapi.PortalMapping, error) {
	hasApp := p.AppID.Valid && p.AppID.String != ""
	hasKeyspace := p.KeyAuthID.Valid && p.KeyAuthID.String != ""

	if hasApp == hasKeyspace {
		return openapi.PortalMapping{Id: "", Type: ""}, fault.New("portal mapping invariant violated",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal(fmt.Sprintf("portal %s has hasApp=%t hasKeyspace=%t", p.ID, hasApp, hasKeyspace)),
			fault.Public("Portal is misconfigured."),
		)
	}

	if hasApp {
		return openapi.PortalMapping{Id: p.AppID.String, Type: openapi.PortalMappingTypeApp}, nil
	}
	return openapi.PortalMapping{Id: p.KeyAuthID.String, Type: openapi.PortalMappingTypeKeyspace}, nil
}

// SameAssociation reports whether two association columns name the same
// resource.
//
// Not raw NullString equality: a row written before these routes existed can
// hold a Valid but empty column, which means "no association" just as NULL does.
// Comparing the structs directly would call those two different and report a
// mapping change for an identical mapping — which, on update, revokes every live
// session for nothing.
func SameAssociation(a sql.NullString, b sql.NullString) bool {
	return associationValue(a) == associationValue(b)
}

func associationValue(c sql.NullString) string {
	if !c.Valid {
		return ""
	}
	return c.String
}

// ErrUnknownMappingType is the single construction for a mapping kind outside
// the enum.
//
// The generated enum makes this unreachable through a well-formed request, but
// every switch over the kind needs a total default, and five copies of the same
// fault chain is five places for the public message to drift.
func ErrUnknownMappingType(mappingType openapi.PortalMappingType) error {
	return fault.New("unknown portal mapping type",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(fmt.Sprintf("unknown mapping type %q", mappingType)),
		fault.Public(ErrMsgInvalidMapping),
	)
}

// ToResponseTolerant maps a stored row for a response that must not fail.
//
// [ToResponse] refuses an ambiguous row because serving one leaves the portal's
// keyspace scope undefined. That refusal is wrong at the end of a mutation: the
// write has already happened, and failing here would roll it back, so a portal
// written before these routes existed could never be disabled. This reports the
// mapping the same way the audit entry does and lets the operator act on the row.
func ToResponseTolerant(p db.Portal) openapi.Portal {
	mappingType, mappingID := DescribeMapping(p)

	var branding *openapi.PortalBranding
	if p.LogoUrl.Valid || p.PrimaryColor.Valid {
		branding = &openapi.PortalBranding{
			LogoUrl:      p.LogoUrl.String,
			PrimaryColor: p.PrimaryColor.String,
		}
	}

	return openapi.Portal{
		Branding:  branding,
		CreatedAt: p.CreatedAt,
		Enabled:   p.Enabled,
		Id:        p.ID,
		Mapping: openapi.PortalMapping{
			Id:   mappingID,
			Type: openapi.PortalMappingType(mappingType),
		},
		Slug:      p.Slug,
		UpdatedAt: p.UpdatedAt.Int64,
	}
}

// DescribeMapping renders a row's mapping for an audit-log entry.
//
// Distinct from [MappingOf], which refuses an ambiguous row because serving one
// would leave a portal's keyspace scope undefined. An audit entry has the
// opposite obligation: a mutation on a row that already violates the invariant
// still has to be recorded, and refusing to describe it would mean refusing to
// delete a misconfigured portal, leaving it with no way out.
//
// So this never fails, and it keeps the two failure modes apart. "invalid" and
// "none" are different problems — two conflicting mappings versus none at all —
// and both ids survive the ambiguous case, because an incident reviewer needs to
// know which resources the row was claiming.
func DescribeMapping(p db.Portal) (mappingType string, mappingID string) {
	hasApp := p.AppID.Valid && p.AppID.String != ""
	hasKeyspace := p.KeyAuthID.Valid && p.KeyAuthID.String != ""

	switch {
	case hasApp && hasKeyspace:
		return "invalid", p.AppID.String + "," + p.KeyAuthID.String
	case hasApp:
		return string(openapi.PortalMappingTypeApp), p.AppID.String
	case hasKeyspace:
		return string(openapi.PortalMappingTypeKeyspace), p.KeyAuthID.String
	default:
		return "none", ""
	}
}

// VerifyMappingOwned reports whether the mapped app or keyspace exists in this
// workspace.
//
// This is the check that keeps a portal from claiming another tenant's resource.
// `idx_app_id` and `idx_key_auth_id` are unique across the whole table, so an
// unvalidated mapping is a permanent global claim: the owning workspace could
// never create its own portal for it, and because the verified-custom-domain
// lookup on the session path is not workspace-scoped, a squatted app id would
// steer session URLs onto the victim's domain.
//
// Runs inside the caller's transaction so the row cannot disappear between this
// check and the write.
func VerifyMappingOwned(ctx context.Context, tx db.DBTX, workspaceID string, m openapi.PortalMapping) error {
	notFound := func(detail string) error {
		return fault.New("portal mapping not found",
			fault.Code(codes.Data.Portal.NotFound.URN()),
			fault.Internal(detail),
			fault.Public(ErrMsgMappingNotFound),
		)
	}

	switch m.Type {
	case openapi.PortalMappingTypeApp:
		_, err := db.Query.FindAppByIdAndWorkspace(ctx, tx, db.FindAppByIdAndWorkspaceParams{
			ID:          m.Id,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return notFound(fmt.Sprintf("app %s is not in workspace %s", m.Id, workspaceID))
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up app"),
				fault.Public("Failed to look up the app."),
			)
		}
		return nil

	case openapi.PortalMappingTypeKeyspace:
		rows, err := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, tx, db.FindKeyAuthsByIdsAndWorkspaceParams{
			WorkspaceID: workspaceID,
			KeyAuthIds:  []string{m.Id},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up keyspace"),
				fault.Public("Failed to look up the keyspace."),
			)
		}
		if len(rows) == 0 {
			return notFound(fmt.Sprintf("keyspace %s is not in workspace %s", m.Id, workspaceID))
		}
		return nil

	default:
		return ErrUnknownMappingType(m.Type)
	}
}

// AuthorizeMappingTarget requires the caller to hold read permission on the
// resource a portal is being pointed at.
//
// [VerifyMappingOwned] asks only whether the resource is in the caller's
// workspace, which is not the same question. Without this, a key holding nothing
// but `update_portal` could re-point a portal at any keyspace in the workspace,
// including ones it has no rights over: the customer's own backend then re-mints
// sessions against the new resource, and that portal's end users see keys the
// remapping key could never have read itself.
//
// The mint path enforces its own ceiling — `authorizeScopes` in
// portal.createSession requires the minting principal to hold the equivalent
// permission on every keyspace the portal resolves to — so this is defence in
// depth rather than the only control. It is deliberately the weakest meaningful
// check, read on the target, because deciding which resource a portal serves is
// not the same authority as granting access to it.
func AuthorizeMappingTarget(
	ctx context.Context,
	tx db.DBTX,
	principal *authprincipal.Principal,
	workspaceID string,
	m openapi.PortalMapping,
) error {
	denied := func(detail string) error {
		// Fresh chain, not a wrap: the rendered query names the api or app id
		// behind the target, which is more than the caller needs in order to learn
		// it is short a grant.
		return fault.New("insufficient permissions for portal mapping target",
			fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
			fault.Internal(detail),
			fault.Public("You do not have permission to point a portal at that resource."),
		)
	}

	switch m.Type {
	case openapi.PortalMappingTypeApp:
		err := principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{ResourceType: rbac.App, ResourceID: "*", Action: rbac.ReadApp}),
			rbac.T(rbac.Tuple{ResourceType: rbac.App, ResourceID: m.Id, Action: rbac.ReadApp}),
		))
		if err != nil {
			return denied(fmt.Sprintf("caller may not read app %s: %s", m.Id, fault.InternalMessage(err)))
		}
		return nil

	case openapi.PortalMappingTypeKeyspace:
		// Keyspace permissions are expressed against the owning api, the same
		// indirection the mint path uses.
		rows, err := db.Query.FindApisByKeyAuthIds(ctx, tx, db.FindApisByKeyAuthIdsParams{
			WorkspaceID: workspaceID,
			KeyAuthIds:  []string{m.Id},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error resolving api for portal keyspace"),
				fault.Public("Failed to look up the keyspace."),
			)
		}
		if len(rows) == 0 || rows[0].ApiID == "" {
			// No owning api, so there is no permission to check against. Refused
			// rather than allowed: an unauthorizable target must not become an
			// unchecked one.
			return denied(fmt.Sprintf("keyspace %s has no owning api in workspace %s", m.Id, workspaceID))
		}

		apiID := rows[0].ApiID
		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.ReadAPI}),
			rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: apiID, Action: rbac.ReadAPI}),
		))
		if err != nil {
			return denied(fmt.Sprintf("caller may not read api %s owning keyspace %s: %s", apiID, m.Id, fault.InternalMessage(err)))
		}
		return nil

	default:
		return ErrUnknownMappingType(m.Type)
	}
}

// ValidateLogoURL rejects anything that is not an absolute https URL within the
// column width.
//
// The value is rendered as an image source in the end-user portal, which has no
// Content-Security-Policy to fall back on. The concrete harm is not script
// execution but that every end user's browser contacts whatever host an operator
// names, handing it their IP and user agent on each page view, plus mixed content
// on a page that displays keys.
func ValidateLogoURL(raw string) error {
	invalid := func(detail string) error {
		return fault.New("invalid logo url",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(detail),
			fault.Public(ErrMsgInvalidLogoURL),
		)
	}

	if raw == "" {
		return invalid("logo url was empty")
	}
	if len(raw) > LogoURLMaxLength {
		return invalid(fmt.Sprintf("logo url is %d characters, limit is %d", len(raw), LogoURLMaxLength))
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return invalid(fmt.Sprintf("logo url is not parseable: %s", err))
	}
	if parsed.Scheme != "https" {
		return invalid(fmt.Sprintf("logo url scheme is %q, expected https", parsed.Scheme))
	}
	if parsed.Host == "" {
		return invalid("logo url has no host")
	}
	return nil
}

// ValidatePrimaryColor requires the six-digit hex form. Three-digit shorthand
// and named colours are rejected so the stored value is always directly usable
// by the renderer.
func ValidatePrimaryColor(raw string) error {
	if hexColor.MatchString(raw) {
		return nil
	}
	return fault.New("invalid primary color",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(fmt.Sprintf("primary color %q is not a six-digit hex colour", raw)),
		fault.Public(ErrMsgInvalidColor),
	)
}

// ToResponse maps a stored row into the public shape.
//
// Branding is present only when at least one branding column is set, so a portal
// with no branding omits the object rather than returning two empty strings.
// There is no name field: the name an operator sees comes from the mapped app or
// keyspace, so that renaming that resource cannot leave a stale copy behind.
func ToResponse(p db.Portal) (openapi.Portal, error) {
	mapping, err := MappingOf(p)
	if err != nil {
		return openapi.Portal{
			Branding:  nil,
			CreatedAt: 0,
			Enabled:   false,
			Id:        "",
			Mapping:   openapi.PortalMapping{Id: "", Type: ""},
			Slug:      "",
			UpdatedAt: 0,
		}, err
	}

	var branding *openapi.PortalBranding
	if p.LogoUrl.Valid || p.PrimaryColor.Valid {
		branding = &openapi.PortalBranding{
			LogoUrl:      p.LogoUrl.String,
			PrimaryColor: p.PrimaryColor.String,
		}
	}

	return openapi.Portal{
		Branding:  branding,
		CreatedAt: p.CreatedAt,
		Enabled:   p.Enabled,
		Id:        p.ID,
		Mapping:   mapping,
		Slug:      p.Slug,
		UpdatedAt: p.UpdatedAt.Int64,
	}, nil
}
