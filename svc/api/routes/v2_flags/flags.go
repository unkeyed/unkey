package handler

import (
	"database/sql"
	"slices"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func workspaceUser(s *zen.Session, admin bool) (*principal.Principal, error) {
	p, err := s.GetPrincipal()
	if err != nil {
		return nil, err
	}
	source, ok := p.Source.(principal.JWTSource)
	if !ok || p.Type != principal.TypeJWT || p.Subject.Type != principal.SubjectTypeUser || p.AuthorizedWorkspaceID == "" {
		return nil, forbidden("A workspace user session is required.")
	}
	if admin && !slices.Contains(source.Roles, "admin") {
		return nil, forbidden("Only workspace admins can change platform features.")
	}
	return p, nil
}

func resolve(flag db.Flag, override sql.NullBool) openapi.WorkspaceFlag {
	value := flag.DefaultValue
	if override.Valid {
		value = override.Bool
	}
	return openapi.WorkspaceFlag{
		Slug: flag.Slug, Description: flag.Description,
		DefaultValue: flag.DefaultValue, Value: value, HasOverride: override.Valid,
		AllowOptIn: flag.AllowOptIn, AllowOptOut: flag.AllowOptOut,
	}
}

func forbidden(message string) error {
	return fault.New("flag access denied", fault.Code(codes.Auth.Authorization.Forbidden.URN()), fault.Public(message))
}
