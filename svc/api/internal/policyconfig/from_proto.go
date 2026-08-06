package policyconfig

import (
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// FromProto is the read-side inverse of setPolicies' mapping:
// stored protos back into response types. Stored data already passed write
// validation, so any unmappable shape (unknown oneof variant, empty oneof,
// a value the response schema cannot express) is corrupt or out-of-band
// state and surfaces as an internal error rather than being silently
// dropped from a list that claims to be complete.
func FromProto(policies []*frontlinev1.Policy) ([]openapi.PolicyResponse, error) {
	out := make([]openapi.PolicyResponse, 0, len(policies))
	for _, p := range policies {
		converted, err := PolicyFromProto(p)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func PolicyFromProto(p *frontlinev1.Policy) (openapi.PolicyResponse, error) {
	out := openapi.PolicyResponse{
		Id:        p.GetId(),
		Name:      p.GetName(),
		Enabled:   p.GetEnabled(),
		Match:     nil,
		Keyauth:   nil,
		Ratelimit: nil,
		Firewall:  nil,
		Openapi:   nil,
		Logging:   nil,
	}

	if len(p.GetMatch()) > 0 {
		match := make([]openapi.MatchExpr, 0, len(p.GetMatch()))
		for _, m := range p.GetMatch() {
			expr, err := mapMatchExprFromProto(m)
			if err != nil {
				return openapi.PolicyResponse{}, err
			}
			match = append(match, expr)
		}
		out.Match = &match
	}

	switch config := p.Config.(type) {
	case *frontlinev1.Policy_Keyauth:
		keyauth, err := mapKeyauthFromProto(p.GetId(), config.Keyauth)
		if err != nil {
			return openapi.PolicyResponse{}, err
		}
		out.Keyauth = keyauth

	case *frontlinev1.Policy_Ratelimit:
		identifier := openapi.RatelimitIdentifier{
			RemoteIp:             nil,
			Header:               nil,
			AuthenticatedSubject: nil,
			Path:                 nil,
			PrincipalField:       nil,
		}
		switch source := config.Ratelimit.GetIdentifier().GetSource().(type) {
		case *frontlinev1.RateLimitIdentifier_RemoteIp:
			identifier.RemoteIp = &openapi.RemoteIpKey{}
		case *frontlinev1.RateLimitIdentifier_Header:
			identifier.Header = &openapi.HeaderKey{Name: source.Header.GetName()}
		case *frontlinev1.RateLimitIdentifier_AuthenticatedSubject:
			identifier.AuthenticatedSubject = &openapi.AuthenticatedSubjectKey{}
		case *frontlinev1.RateLimitIdentifier_Path:
			identifier.Path = &openapi.PathKey{}
		case *frontlinev1.RateLimitIdentifier_PrincipalField:
			identifier.PrincipalField = &openapi.PrincipalFieldKey{Path: source.PrincipalField.GetPath()}
		default:
			return openapi.PolicyResponse{}, unmappable(p.GetId(), "ratelimit identifier")
		}
		out.Ratelimit = &openapi.RatelimitPolicy{
			Limit:      config.Ratelimit.GetLimit(),
			WindowMs:   config.Ratelimit.GetWindowMs(),
			Identifier: identifier,
		}

	case *frontlinev1.Policy_Firewall:
		out.Firewall = &openapi.FirewallPolicy{
			Action: openapi.FirewallPolicyAction(config.Firewall.GetAction().String()),
		}

	case *frontlinev1.Policy_Openapi:
		out.Openapi = ptr.P(openapi.OpenapiPolicy{})

	case *frontlinev1.Policy_Logging:
		out.Logging = &openapi.LoggingPolicy{
			RequestHeaders:  ptr.P(config.Logging.GetRequestHeaders()),
			ResponseHeaders: ptr.P(config.Logging.GetResponseHeaders()),
			RequestBody:     ptr.P(config.Logging.GetRequestBody()),
			ResponseBody:    ptr.P(config.Logging.GetResponseBody()),
		}

	default:
		return openapi.PolicyResponse{}, unmappable(p.GetId(), "config variant")
	}

	return out, nil
}

func mapKeyauthFromProto(policyID string, k *frontlinev1.KeyAuth) (*openapi.KeyauthPolicy, error) {
	// The response schema requires at least one keyspace, matching write
	// validation; a keyspace-less keyauth cannot come from our writers.
	if len(k.GetKeySpaceIds()) == 0 {
		return nil, unmappable(policyID, "keyauth without keyspaces")
	}

	out := &openapi.KeyauthPolicy{
		Keyspaces:       k.GetKeySpaceIds(),
		PermissionQuery: k.PermissionQuery,
		Locations:       nil,
		Ratelimits:      nil,
	}

	if len(k.GetLocations()) > 0 {
		locations := make([]openapi.KeyLocation, 0, len(k.GetLocations()))
		for _, loc := range k.GetLocations() {
			mapped := openapi.KeyLocation{
				Bearer:     nil,
				Header:     nil,
				QueryParam: nil,
			}
			switch location := loc.GetLocation().(type) {
			case *frontlinev1.KeyLocation_Bearer:
				mapped.Bearer = &openapi.BearerTokenLocation{}
			case *frontlinev1.KeyLocation_Header:
				mapped.Header = &openapi.HeaderKeyLocation{Name: location.Header.GetName(), StripPrefix: nil}
				if prefix := location.Header.GetStripPrefix(); prefix != "" {
					mapped.Header.StripPrefix = &prefix
				}
			case *frontlinev1.KeyLocation_QueryParam:
				mapped.QueryParam = &openapi.QueryParamKeyLocation{Name: location.QueryParam.GetName()}
			default:
				return nil, unmappable(policyID, "key location")
			}
			locations = append(locations, mapped)
		}
		out.Locations = &locations
	}

	if len(k.GetRatelimits()) > 0 {
		ratelimits := make([]openapi.KeyRatelimit, 0, len(k.GetRatelimits()))
		for _, rl := range k.GetRatelimits() {
			ratelimits = append(ratelimits, openapi.KeyRatelimit{
				Name:     rl.GetName(),
				Limit:    rl.Limit,
				Duration: rl.Duration,
				Cost:     rl.Cost,
			})
		}
		out.Ratelimits = &ratelimits
	}

	return out, nil
}

func mapMatchExprFromProto(m *frontlinev1.MatchExpr) (openapi.MatchExpr, error) {
	out := openapi.MatchExpr{
		Path:       nil,
		Method:     nil,
		Header:     nil,
		QueryParam: nil,
	}

	switch expr := m.GetExpr().(type) {
	case *frontlinev1.MatchExpr_Path:
		sm, err := mapStringMatchFromProto(expr.Path.GetPath())
		if err != nil {
			return out, err
		}
		out.Path = &openapi.PathMatch{Path: sm}

	case *frontlinev1.MatchExpr_Method:
		methods := make([]openapi.MethodMatchMethods, 0, len(expr.Method.GetMethods()))
		for _, method := range expr.Method.GetMethods() {
			methods = append(methods, openapi.MethodMatchMethods(method))
		}
		out.Method = &openapi.MethodMatch{Methods: methods}

	case *frontlinev1.MatchExpr_Header:
		field := openapi.FieldMatch{Name: expr.Header.GetName(), Present: nil, Value: nil}
		switch match := expr.Header.GetMatch().(type) {
		case *frontlinev1.HeaderMatch_Present:
			// The schema only admits `present: true`; the gateway's
			// absent-match (present=false) has no response representation.
			if !match.Present {
				return out, unmappable("", "header absent-match")
			}
			field.Present = ptr.P(openapi.FieldMatchPresent(match.Present))
		case *frontlinev1.HeaderMatch_Value:
			sm, err := mapStringMatchFromProto(match.Value)
			if err != nil {
				return out, err
			}
			field.Value = &sm
		default:
			return out, unmappable("", "header match")
		}
		out.Header = &field

	case *frontlinev1.MatchExpr_QueryParam:
		field := openapi.FieldMatch{Name: expr.QueryParam.GetName(), Present: nil, Value: nil}
		switch match := expr.QueryParam.GetMatch().(type) {
		case *frontlinev1.QueryParamMatch_Present:
			if !match.Present {
				return out, unmappable("", "queryParam absent-match")
			}
			field.Present = ptr.P(openapi.FieldMatchPresent(match.Present))
		case *frontlinev1.QueryParamMatch_Value:
			sm, err := mapStringMatchFromProto(match.Value)
			if err != nil {
				return out, err
			}
			field.Value = &sm
		default:
			return out, unmappable("", "queryParam match")
		}
		out.QueryParam = &field

	default:
		return out, unmappable("", "match expression")
	}

	return out, nil
}

func mapStringMatchFromProto(s *frontlinev1.StringMatch) (openapi.StringMatch, error) {
	out := openapi.StringMatch{
		Exact:      nil,
		Prefix:     nil,
		Regex:      nil,
		IgnoreCase: nil,
	}
	// Proto's zero value and an absent flag are indistinguishable; omit
	// false so responses stay minimal and roundtrip-stable with setPolicies.
	if s.GetIgnoreCase() {
		out.IgnoreCase = ptr.P(true)
	}
	switch match := s.GetMatch().(type) {
	case *frontlinev1.StringMatch_Exact:
		out.Exact = &match.Exact
	case *frontlinev1.StringMatch_Prefix:
		out.Prefix = &match.Prefix
	case *frontlinev1.StringMatch_Regex:
		out.Regex = &match.Regex
	default:
		return out, unmappable("", "string match")
	}
	return out, nil
}

func unmappable(policyID, what string) error {
	internal := "stored policy has an unmappable " + what
	if policyID != "" {
		internal += " (policy " + policyID + ")"
	}
	return fault.New(
		"unmappable stored policy",
		fault.Code(codes.App.Internal.UnexpectedError.URN()),
		fault.Internal(internal),
		fault.Public("We're unable to read the stored policies."),
	)
}
