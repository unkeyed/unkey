package policyconfig

import (
	"fmt"
	"regexp"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

// ToProto parses request policies into the protos frontline evaluates,
// generating an id per policy. Conversion is also the validation pass: it
// enforces the rules the OpenAPI schema cannot express (exactly-one variants,
// valid regex and permission queries), and its errors are user-facing,
// naming the offending field.
func ToProto(policies []openapi.Policy) ([]*frontlinev1.Policy, error) {
	out := make([]*frontlinev1.Policy, 0, len(policies))
	for i, p := range policies {
		converted, err := PolicyToProto(fmt.Sprintf("policies[%d]", i), p)
		if err != nil {
			return nil, err
		}
		converted.Id = uid.New(uid.PolicyPrefix)
		out = append(out, converted)
	}
	return out, nil
}

// VariantName names the set variant for audit metadata.
func VariantName(p *frontlinev1.Policy) string {
	switch p.Config.(type) {
	case *frontlinev1.Policy_Keyauth:
		return "keyauth"
	case *frontlinev1.Policy_Ratelimit:
		return "ratelimit"
	case *frontlinev1.Policy_Firewall:
		return "firewall"
	case *frontlinev1.Policy_Openapi:
		return "openapi"
	default:
		return "unknown"
	}
}

// PolicyToProto converts and validates a single policy. It leaves Id unset;
// callers own identity (ToProto generates fresh ids, updatePolicy keeps the
// stored one).
func PolicyToProto(path string, p openapi.Policy) (*frontlinev1.Policy, error) {
	if err := exactlyOne(path, "keyauth, ratelimit, firewall or openapi",
		p.Keyauth != nil, p.Ratelimit != nil, p.Firewall != nil, p.Openapi != nil); err != nil {
		return nil, err
	}

	out := &frontlinev1.Policy{
		Name:    p.Name,
		Enabled: proto.Bool(p.Enabled),
	}

	// match is a sibling of the oneof, present on every variant.
	for i, m := range ptr.SafeDeref(p.Match) {
		expr, err := mapMatchExprToProto(fmt.Sprintf("%s.match[%d]", path, i), m)
		if err != nil {
			return nil, err
		}
		out.Match = append(out.Match, expr)
	}

	switch {
	case p.Keyauth != nil:
		keyauth, err := mapKeyauthToProto(path+".keyauth", *p.Keyauth)
		if err != nil {
			return nil, err
		}
		out.Config = &frontlinev1.Policy_Keyauth{Keyauth: keyauth}

	case p.Ratelimit != nil:
		ratelimit, err := mapRatelimitToProto(path+".ratelimit", *p.Ratelimit)
		if err != nil {
			return nil, err
		}
		out.Config = &frontlinev1.Policy_Ratelimit{Ratelimit: ratelimit}

	case p.Firewall != nil:
		action, ok := frontlinev1.Action_value[string(p.Firewall.Action)]
		if !ok {
			return nil, invalid(fmt.Sprintf("%s.firewall.action %q is not a known action.", path, p.Firewall.Action))
		}
		out.Config = &frontlinev1.Policy_Firewall{Firewall: &frontlinev1.Firewall{Action: frontlinev1.Action(action)}}

	case p.Openapi != nil:
		out.Config = &frontlinev1.Policy_Openapi{Openapi: &frontlinev1.OpenApiRequestValidation{}}
	}

	return out, nil
}

// mapRatelimitToProto validates the single/compound identifier duality:
// exactly one of identifier or identifiers must be set. The single form maps
// to the proto identifier field, the compound form to the repeated
// identifiers field, so responses can render policies in the shape they were
// written.
func mapRatelimitToProto(path string, r openapi.RatelimitPolicy) (*frontlinev1.RateLimit, error) {
	hasList := r.Identifiers != nil && len(*r.Identifiers) > 0
	if err := exactlyOne(path, "identifier or identifiers", r.Identifier != nil, hasList); err != nil {
		return nil, err
	}

	out := &frontlinev1.RateLimit{
		Limit:    r.Limit,
		WindowMs: r.WindowMs,
	}

	if r.Identifier != nil {
		identifier, err := mapRatelimitIdentifierToProto(path+".identifier", *r.Identifier)
		if err != nil {
			return nil, err
		}
		out.Identifier = identifier
		return out, nil
	}

	if len(*r.Identifiers) > maxCompoundIdentifiers {
		return nil, invalid(fmt.Sprintf("%s.identifiers must not have more than %d entries.", path, maxCompoundIdentifiers))
	}
	for i, id := range *r.Identifiers {
		identifier, err := mapRatelimitIdentifierToProto(fmt.Sprintf("%s.identifiers[%d]", path, i), id)
		if err != nil {
			return nil, err
		}
		out.Identifiers = append(out.Identifiers, identifier)
	}
	return out, nil
}

// maxCompoundIdentifiers caps the dimensions of a compound rate limit key.
// Mirrors the OpenAPI schema's maxItems; enforced here too because the
// conversion pass is the validation layer for anything callers bypass.
const maxCompoundIdentifiers = 5

func mapRatelimitIdentifierToProto(path string, id openapi.RatelimitIdentifier) (*frontlinev1.RateLimitIdentifier, error) {
	if err := exactlyOne(path, "remoteIp, header, authenticatedSubject, path or principalField",
		id.RemoteIp != nil, id.Header != nil, id.AuthenticatedSubject != nil, id.Path != nil, id.PrincipalField != nil); err != nil {
		return nil, err
	}
	identifier := &frontlinev1.RateLimitIdentifier{}
	switch {
	case id.RemoteIp != nil:
		identifier.Source = &frontlinev1.RateLimitIdentifier_RemoteIp{RemoteIp: &frontlinev1.RemoteIpKey{}}
	case id.Header != nil:
		identifier.Source = &frontlinev1.RateLimitIdentifier_Header{Header: &frontlinev1.HeaderKey{Name: id.Header.Name}}
	case id.AuthenticatedSubject != nil:
		identifier.Source = &frontlinev1.RateLimitIdentifier_AuthenticatedSubject{AuthenticatedSubject: &frontlinev1.AuthenticatedSubjectKey{}}
	case id.Path != nil:
		identifier.Source = &frontlinev1.RateLimitIdentifier_Path{Path: &frontlinev1.PathKey{}}
	case id.PrincipalField != nil:
		identifier.Source = &frontlinev1.RateLimitIdentifier_PrincipalField{PrincipalField: &frontlinev1.PrincipalFieldKey{Path: id.PrincipalField.Path}}
	}
	return identifier, nil
}

func mapKeyauthToProto(path string, k openapi.KeyauthPolicy) (*frontlinev1.KeyAuth, error) {
	out := &frontlinev1.KeyAuth{
		KeySpaceIds:     k.Keyspaces,
		PermissionQuery: k.PermissionQuery,
	}

	for i, loc := range ptr.SafeDeref(k.Locations) {
		if err := exactlyOne(fmt.Sprintf("%s.locations[%d]", path, i), "bearer, header or queryParam",
			loc.Bearer != nil, loc.Header != nil, loc.QueryParam != nil); err != nil {
			return nil, err
		}
		protoLoc := &frontlinev1.KeyLocation{}
		switch {
		case loc.Bearer != nil:
			protoLoc.Location = &frontlinev1.KeyLocation_Bearer{Bearer: &frontlinev1.BearerTokenLocation{}}
		case loc.Header != nil:
			protoLoc.Location = &frontlinev1.KeyLocation_Header{Header: &frontlinev1.HeaderKeyLocation{
				Name:        loc.Header.Name,
				StripPrefix: ptr.SafeDeref(loc.Header.StripPrefix),
			}}
		case loc.QueryParam != nil:
			protoLoc.Location = &frontlinev1.KeyLocation_QueryParam{QueryParam: &frontlinev1.QueryParamKeyLocation{Name: loc.QueryParam.Name}}
		}
		out.Locations = append(out.Locations, protoLoc)
	}

	for i, rl := range ptr.SafeDeref(k.Ratelimits) {
		if (rl.Limit == nil) != (rl.Duration == nil) {
			return nil, invalid(fmt.Sprintf("%s.ratelimits[%d] must set limit and duration together.", path, i))
		}
		out.Ratelimits = append(out.Ratelimits, &frontlinev1.KeyRatelimit{
			Name:     rl.Name,
			Limit:    rl.Limit,
			Duration: rl.Duration,
			Cost:     rl.Cost,
		})
	}

	// The gateway parses this per request (frontline keyauth executor) and
	// fails every matching request on a bad query, so reject it at write
	// time with the same parser. Empty means no permission check.
	if pq := ptr.SafeDeref(k.PermissionQuery); pq != "" {
		if _, err := rbac.ParseQuery(pq); err != nil {
			return nil, invalid(fmt.Sprintf("%s.permissionQuery is not a valid permission query: %s", path, err))
		}
	}

	return out, nil
}

func mapMatchExprToProto(path string, m openapi.MatchExpr) (*frontlinev1.MatchExpr, error) {
	if err := exactlyOne(path, "path, method, header or queryParam",
		m.Path != nil, m.Method != nil, m.Header != nil, m.QueryParam != nil); err != nil {
		return nil, err
	}

	switch {
	case m.Path != nil:
		sm, err := mapStringMatchToProto(path+".path.path", m.Path.Path)
		if err != nil {
			return nil, err
		}
		return &frontlinev1.MatchExpr{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{Path: sm}}}, nil

	case m.Method != nil:
		methods := make([]string, 0, len(m.Method.Methods))
		for _, method := range m.Method.Methods {
			methods = append(methods, string(method))
		}
		return &frontlinev1.MatchExpr{Expr: &frontlinev1.MatchExpr_Method{Method: &frontlinev1.MethodMatch{Methods: methods}}}, nil

	case m.Header != nil:
		header := &frontlinev1.HeaderMatch{Name: m.Header.Name}
		if err := exactlyOne(path+".header", "present or value",
			m.Header.Present != nil, m.Header.Value != nil); err != nil {
			return nil, err
		}
		if m.Header.Present != nil {
			header.Match = &frontlinev1.HeaderMatch_Present{Present: bool(*m.Header.Present)}
		} else {
			sm, err := mapStringMatchToProto(path+".header.value", *m.Header.Value)
			if err != nil {
				return nil, err
			}
			header.Match = &frontlinev1.HeaderMatch_Value{Value: sm}
		}
		return &frontlinev1.MatchExpr{Expr: &frontlinev1.MatchExpr_Header{Header: header}}, nil

	default: // m.QueryParam != nil, guaranteed by exactlyOne above
		queryParam := &frontlinev1.QueryParamMatch{Name: m.QueryParam.Name}
		if err := exactlyOne(path+".queryParam", "present or value",
			m.QueryParam.Present != nil, m.QueryParam.Value != nil); err != nil {
			return nil, err
		}
		if m.QueryParam.Present != nil {
			queryParam.Match = &frontlinev1.QueryParamMatch_Present{Present: bool(*m.QueryParam.Present)}
		} else {
			sm, err := mapStringMatchToProto(path+".queryParam.value", *m.QueryParam.Value)
			if err != nil {
				return nil, err
			}
			queryParam.Match = &frontlinev1.QueryParamMatch_Value{Value: sm}
		}
		return &frontlinev1.MatchExpr{Expr: &frontlinev1.MatchExpr_QueryParam{QueryParam: queryParam}}, nil
	}
}

func mapStringMatchToProto(path string, s openapi.StringMatch) (*frontlinev1.StringMatch, error) {
	if err := exactlyOne(path, "exact, prefix or regex",
		s.Exact != nil, s.Prefix != nil, s.Regex != nil); err != nil {
		return nil, err
	}

	out := &frontlinev1.StringMatch{IgnoreCase: ptr.SafeDeref(s.IgnoreCase)}
	switch {
	case s.Exact != nil:
		out.Match = &frontlinev1.StringMatch_Exact{Exact: *s.Exact}
	case s.Prefix != nil:
		out.Match = &frontlinev1.StringMatch_Prefix{Prefix: *s.Prefix}
	case s.Regex != nil:
		if _, err := regexp.Compile(*s.Regex); err != nil {
			return nil, invalid(fmt.Sprintf("%s.regex is not a valid regular expression: %s", path, err))
		}
		out.Match = &frontlinev1.StringMatch_Regex{Regex: *s.Regex}
	}
	return out, nil
}

// exactlyOne mirrors a proto oneof: exactly one variant must be set.
// variants renders verbatim into the error message.
func exactlyOne(path, variants string, set ...bool) error {
	n := 0
	for _, s := range set {
		if s {
			n++
		}
	}
	if n == 1 {
		return nil
	}
	detail := "none are set"
	if n > 1 {
		detail = fmt.Sprintf("%d are set", n)
	}
	return invalid(fmt.Sprintf("%s must set exactly one of %s; %s.", path, variants, detail))
}

func invalid(message string) error {
	return fault.New(
		"invalid policy",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("policy validation failed"),
		fault.Public(message),
	)
}
