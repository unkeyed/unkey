// Package portalconfig holds helpers shared by the operator-facing
// portal-configuration routes (create/list/update/delete). It centralizes the
// two pieces of logic those handlers would otherwise duplicate: enforcing that a
// configuration maps to exactly one keyspace or app, and mapping a stored
// configuration (plus its branding) into the public API shape.
package portalconfig

import (
	"database/sql"
	"fmt"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// ErrMsgInvalidMapping is the public message returned when a request does not
// map a configuration to exactly one keyspace or app.
const ErrMsgInvalidMapping = "A portal configuration must map to exactly one keyspace (keyspaceId) or one app (appId)."

// Mapping validates that exactly one of keyspaceId or appId is provided and
// returns the corresponding nullable columns. A configuration maps to exactly
// one keyspace or one app; neither or both is a client error.
func Mapping(appID, keyspaceID *string) (appCol sql.NullString, keyAuthCol sql.NullString, err error) {
	hasApp := appID != nil && *appID != ""
	hasKeyspace := keyspaceID != nil && *keyspaceID != ""

	if hasApp == hasKeyspace {
		return sql.NullString{}, sql.NullString{}, fault.New("invalid portal config mapping",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("expected exactly one of appId/keyspaceId (hasApp=%t, hasKeyspace=%t)", hasApp, hasKeyspace)),
			fault.Public(ErrMsgInvalidMapping),
		)
	}

	if hasApp {
		return sql.NullString{String: *appID, Valid: true}, sql.NullString{}, nil
	}
	return sql.NullString{}, sql.NullString{String: *keyspaceID, Valid: true}, nil
}

// ToResponse maps a stored configuration and its (possibly absent) branding
// columns into the public API shape. Branding is included only when at least one
// branding column is set.
func ToResponse(cfg db.PortalConfiguration, logoUrl, primaryColor sql.NullString) openapi.PortalConfiguration {
	// A nil-valued sql.NullString/NullInt64 already carries the "" / 0 zero
	// value, which maps to the omitempty JSON fields, so the columns can be read
	// directly into a single complete literal.
	var branding *openapi.PortalBranding
	if logoUrl.Valid || primaryColor.Valid {
		branding = &openapi.PortalBranding{
			LogoUrl:      logoUrl.String,
			PrimaryColor: primaryColor.String,
		}
	}

	return openapi.PortalConfiguration{
		Id:          cfg.ID,
		Slug:        cfg.Slug,
		DisplayName: cfg.DisplayName,
		AppId:       cfg.AppID.String,
		KeyspaceId:  cfg.KeyAuthID.String,
		Enabled:     cfg.Enabled,
		ReturnUrl:   cfg.ReturnUrl.String,
		Branding:    branding,
		CreatedAt:   cfg.CreatedAt,
		UpdatedAt:   cfg.UpdatedAt.Int64,
	}
}
