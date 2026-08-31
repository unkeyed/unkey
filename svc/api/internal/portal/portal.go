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
	"strings"

	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Public messages. A mapping that names a resource the caller does not own is
// deliberately reported as a plain not-found, identical to naming one that does
// not exist anywhere: a caller must not be able to use the difference to learn
// that another workspace holds it.
const (
	ErrMsgInvalidMapping   = "A portal must map to exactly one app or one keyspace."
	ErrMsgMappingNotFound  = "The app or keyspace was not found."
	ErrMsgInvalidLogoURL   = "logoUrl must be an absolute https:// URL of at most 500 characters."
	ErrMsgInvalidColor     = "primaryColor must be a six-digit hex colour, for example #6366f1."
	ErrMsgInvalidReturnURL = "returnUrl must be an absolute https:// URL of at most 500 characters."
)

// LogoURLMaxLength matches the column width. Enforced here so an over-long value
// is a validation error rather than a truncation or a driver error surfacing as a
// 500.
const LogoURLMaxLength = 500

// ReturnURLMaxLength matches the portal_sessions.return_url column width.
const ReturnURLMaxLength = 500

// MappingType names the kind of resource a portal serves.
type MappingType string

const (
	MappingTypeApp      MappingType = "app"
	MappingTypeKeyspace MappingType = "keyspace"
)

// Mapping is the single resource a portal serves keys for.
//
// The wire format carries two mutually exclusive optional ids (`keyspaceId` and
// `appId`), which can express both and neither. This type cannot: every value
// that exists has been through [MappingFrom], so once a handler holds one the
// invariant is already established and no downstream code re-checks it.
type Mapping struct {
	Type MappingType
	ID   string
}

// MappingFrom parses the flat wire pair into the domain type.
//
// This is the boundary. `oneOf` in the spec rejects both-or-neither for a
// well-formed request, but it cannot express that a whitespace-only id is not an
// id, and a client can always send past a generated enum, so the check is here
// too rather than trusted from the schema.
func MappingFrom(keyspaceID *string, appID *string) (Mapping, error) {
	ks := ""
	if keyspaceID != nil {
		ks = strings.TrimSpace(*keyspaceID)
	}
	app := ""
	if appID != nil {
		app = strings.TrimSpace(*appID)
	}

	switch {
	case ks != "" && app != "":
		return Mapping{Type: "", ID: ""}, fault.New("portal names two resources",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("both keyspaceId and appId were provided"),
			fault.Public(ErrMsgInvalidMapping),
		)
	case ks != "":
		return Mapping{Type: MappingTypeKeyspace, ID: ks}, nil
	case app != "":
		return Mapping{Type: MappingTypeApp, ID: app}, nil
	default:
		return Mapping{Type: "", ID: ""}, fault.New("portal names no resource",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("neither keyspaceId nor appId was provided"),
			fault.Public(ErrMsgInvalidMapping),
		)
	}
}

// ColumnsFor turns a mapping into the two nullable columns that carry it.
// Exactly one is set.
//
// [MappingFrom] already established that the id is non-empty and the kind is
// known, so the only remaining case is a zero value a caller built by hand.
func ColumnsFor(m Mapping) (appID sql.NullString, keyAuthID sql.NullString, err error) {
	switch m.Type {
	case MappingTypeApp:
		return sql.NullString{String: m.ID, Valid: true}, sql.NullString{}, nil
	case MappingTypeKeyspace:
		return sql.NullString{}, sql.NullString{String: m.ID, Valid: true}, nil
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
func MappingOf(p db.Portal) (Mapping, error) {
	hasApp := p.AppID.Valid && p.AppID.String != ""
	hasKeyspace := p.KeyAuthID.Valid && p.KeyAuthID.String != ""

	if hasApp == hasKeyspace {
		return Mapping{Type: "", ID: ""}, fault.New("portal mapping invariant violated",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal(fmt.Sprintf("portal %s has hasApp=%t hasKeyspace=%t", p.ID, hasApp, hasKeyspace)),
			fault.Public("Portal is misconfigured."),
		)
	}

	if hasApp {
		return Mapping{Type: MappingTypeApp, ID: p.AppID.String}, nil
	}
	return Mapping{Type: MappingTypeKeyspace, ID: p.KeyAuthID.String}, nil
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
func ErrUnknownMappingType(mappingType MappingType) error {
	return fault.New("unknown portal mapping type",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(fmt.Sprintf("unknown mapping type %q", mappingType)),
		fault.Public(ErrMsgInvalidMapping),
	)
}

// responseIDs renders a mapping as the flat wire pair. Exactly one is non-nil,
// which is what lets a reader treat the present one as the resource served.
func responseIDs(m Mapping) (keyspaceID *openapi.PortalKeyspaceId, appID *openapi.PortalAppId) {
	switch m.Type {
	case MappingTypeApp:
		id := openapi.PortalAppId(m.ID)
		return nil, &id
	case MappingTypeKeyspace:
		id := openapi.PortalKeyspaceId(m.ID)
		return &id, nil
	default:
		return nil, nil
	}
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

	keyspaceID, appID := responseIDs(Mapping{Type: MappingType(mappingType), ID: mappingID})

	return openapi.Portal{
		AppId:       appID,
		Branding:    branding,
		CreatedAt:   p.CreatedAt,
		DisplayName: p.DisplayName,
		Enabled:     p.Enabled,
		Id:          p.ID,
		KeyspaceId:  keyspaceID,
		Slug:        p.Slug,
		UpdatedAt:   p.UpdatedAt.Int64,
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
		return string(MappingTypeApp), p.AppID.String
	case hasKeyspace:
		return string(MappingTypeKeyspace), p.KeyAuthID.String
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
func VerifyMappingOwned(ctx context.Context, tx db.DBTX, workspaceID string, m Mapping) error {
	notFound := func(detail string) error {
		return fault.New("portal mapping not found",
			fault.Code(codes.Data.Portal.NotFound.URN()),
			fault.Internal(detail),
			fault.Public(ErrMsgMappingNotFound),
		)
	}

	switch m.Type {
	case MappingTypeApp:
		_, err := db.Query.FindAppByIdAndWorkspace(ctx, tx, db.FindAppByIdAndWorkspaceParams{
			ID:          m.ID,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return notFound(fmt.Sprintf("app %s is not in workspace %s", m.ID, workspaceID))
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up app"),
				fault.Public("Failed to look up the app."),
			)
		}
		return nil

	case MappingTypeKeyspace:
		rows, err := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, tx, db.FindKeyAuthsByIdsAndWorkspaceParams{
			WorkspaceID: workspaceID,
			KeyAuthIds:  []string{m.ID},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up keyspace"),
				fault.Public("Failed to look up the keyspace."),
			)
		}
		if len(rows) == 0 {
			return notFound(fmt.Sprintf("keyspace %s is not in workspace %s", m.ID, workspaceID))
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
	m Mapping,
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
	case MappingTypeApp:
		// Legacy tuples only, unlike the keyspace arm below. An app URN is
		// addressed as projects/{project_id}/apps/{app_id} and this function is
		// handed an app id alone, and there is no ReadApp action in
		// pkg/rbac/permissions to pair with it. Both belong to migrating
		// v2_apps_get_app, which authorizes the same way. Until then a portal can
		// only be pointed at an app by a caller holding a legacy grant, which is
		// no worse than reading the app itself.
		err := principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{ResourceType: rbac.App, ResourceID: "*", Action: rbac.ReadApp}),
			rbac.T(rbac.Tuple{ResourceType: rbac.App, ResourceID: m.ID, Action: rbac.ReadApp}),
		))
		if err != nil {
			return denied(fmt.Sprintf("caller may not read app %s: %s", m.ID, fault.InternalMessage(err)))
		}
		return nil

	case MappingTypeKeyspace:
		// Keyspace permissions are expressed against the owning api, the same
		// indirection the mint path uses.
		rows, err := db.Query.FindApisByKeyAuthIds(ctx, tx, db.FindApisByKeyAuthIdsParams{
			WorkspaceID: workspaceID,
			KeyAuthIds:  []string{m.ID},
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
			return denied(fmt.Sprintf("keyspace %s has no owning api in workspace %s", m.ID, workspaceID))
		}

		apiID := rows[0].ApiID
		// The URN arm is what makes this reachable from the dashboard. A
		// dashboard operator authenticates with a JWT whose only grant is the
		// workspace-wide admin URN — `admin:*` becomes `unkey:v1:{ws}:**#*`,
		// minted by the proxy locally and by the WorkOS permission translator in
		// production, neither of which emits a legacy tuple. URN wildcards expand
		// for URN queries only, so a tuple-only check can never pass for the one
		// caller this operator route exists to serve.
		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.ReadAPI}),
			rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: apiID, Action: rbac.ReadAPI}),
			rbac.U(
				urn.New().Workspace(workspaceID).Project(rows[0].ProjectID).Keyspace(m.ID),
				permissions.Read{},
			),
		))
		if err != nil {
			return denied(fmt.Sprintf("caller may not read api %s owning keyspace %s: %s", apiID, m.ID, fault.InternalMessage(err)))
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
	return validateHTTPSURL(raw, LogoURLMaxLength, "logo url", ErrMsgInvalidLogoURL)
}

// ValidateReturnURL rejects anything that is not an absolute https URL within the
// column width.
//
// Stricter stakes than the logo. This value is rendered as an anchor href in the
// end-user portal, so a `javascript:` scheme executes in the end user's browser
// on click, with the portal's own origin -- it could call the portal API as that
// user. The OpenAPI `format: uri` on the field does not help: `javascript:...` is
// a perfectly valid URI, and the validator does not assert formats anyway.
func ValidateReturnURL(raw string) error {
	return validateHTTPSURL(raw, ReturnURLMaxLength, "return url", ErrMsgInvalidReturnURL)
}

// validateHTTPSURL is the shared check behind both. Absolute https only: a
// scheme-relative or path-only value would resolve against the portal's own
// origin, and every other scheme is either inert (mailto), unencrypted (http),
// or executable (javascript, data).
func validateHTTPSURL(raw string, maxLength int, label string, publicMessage string) error {
	invalid := func(detail string) error {
		return fault.New(fmt.Sprintf("invalid %s", label),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(detail),
			fault.Public(publicMessage),
		)
	}

	if raw == "" {
		return invalid(fmt.Sprintf("%s was empty", label))
	}
	if len(raw) > maxLength {
		return invalid(fmt.Sprintf("%s is %d characters, limit is %d", label, len(raw), maxLength))
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return invalid(fmt.Sprintf("%s is not parseable: %s", label, err))
	}
	if parsed.Scheme != "https" {
		return invalid(fmt.Sprintf("%s scheme is %q, expected https", label, parsed.Scheme))
	}
	if parsed.Host == "" {
		return invalid(fmt.Sprintf("%s has no host", label))
	}
	return nil
}

// ToResponse maps a stored row into the public shape.
//
// Branding is present only when at least one branding column is set, so a portal
// with no branding omits the object rather than returning two empty strings.
func ToResponse(p db.Portal) (openapi.Portal, error) {
	mapping, err := MappingOf(p)
	if err != nil {
		return openapi.Portal{
			AppId:       nil,
			Branding:    nil,
			CreatedAt:   0,
			DisplayName: "",
			Enabled:     false,
			Id:          "",
			KeyspaceId:  nil,
			Slug:        "",
			UpdatedAt:   0,
		}, err
	}

	var branding *openapi.PortalBranding
	if p.LogoUrl.Valid || p.PrimaryColor.Valid {
		branding = &openapi.PortalBranding{
			LogoUrl:      p.LogoUrl.String,
			PrimaryColor: p.PrimaryColor.String,
		}
	}

	keyspaceID, appID := responseIDs(mapping)

	return openapi.Portal{
		AppId:       appID,
		Branding:    branding,
		CreatedAt:   p.CreatedAt,
		DisplayName: p.DisplayName,
		Enabled:     p.Enabled,
		Id:          p.ID,
		KeyspaceId:  keyspaceID,
		Slug:        p.Slug,
		UpdatedAt:   p.UpdatedAt.Int64,
	}, nil
}
