package handler

import (
	"bytes"
	"encoding/json"
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
		return nil, forbidden("Only workspace admins can change flag overrides.")
	}
	return p, nil
}

func resolve(flag db.Flag, override json.RawMessage) (openapi.WorkspaceFlag, error) {
	defaultValue, err := parseValue(flag.Type, flag.DefaultValue)
	if err != nil {
		return openapi.WorkspaceFlag{}, fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()))
	}
	value := defaultValue
	if override != nil {
		value, err = parseValue(flag.Type, override)
		if err != nil {
			return openapi.WorkspaceFlag{}, fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}
	}
	return openapi.WorkspaceFlag{
		Slug: flag.Slug, Description: flag.Description, Type: openapi.WorkspaceFlagType(flag.Type),
		DefaultValue: defaultValue, Value: value, HasOverride: override != nil,
		AllowOptIn: flag.AllowOptIn, AllowOptOut: flag.AllowOptOut,
	}, nil
}

func parseValue(kind db.FlagsType, raw json.RawMessage) (openapi.FlagValue, error) {
	var value openapi.FlagValue
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return value, invalidValue()
	}
	var err error
	switch kind {
	case db.FlagsTypeBoolean:
		var decoded bool
		err = json.Unmarshal(raw, &decoded)
	case db.FlagsTypeString:
		var decoded string
		err = json.Unmarshal(raw, &decoded)
	case db.FlagsTypeNumber:
		var decoded float64
		err = json.Unmarshal(raw, &decoded)
	default:
		return value, invalidValue()
	}
	if err != nil {
		return value, invalidValue()
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, err
	}
	return value, nil
}

func invalidValue() error {
	return fault.New("invalid flag value", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Value must be a non-null scalar matching the flag type."))
}

func forbidden(message string) error {
	return fault.New("flag access denied", fault.Code(codes.Auth.Authorization.Forbidden.URN()), fault.Public(message))
}
