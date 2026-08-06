package policyconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

func TestMapPolicyFromProtoVariants(t *testing.T) {
	t.Run("keyauth with all fields", func(t *testing.T) {
		got, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_KEBAP1234",
			Name:    "keyauth",
			Enabled: proto.Bool(true),
			Config: &frontlinev1.Policy_Keyauth{Keyauth: &frontlinev1.KeyAuth{
				KeySpaceIds:     []string{"ks_1", "ks_2"},
				PermissionQuery: proto.String("documents.read"),
				Locations: []*frontlinev1.KeyLocation{
					{Location: &frontlinev1.KeyLocation_Bearer{Bearer: &frontlinev1.BearerTokenLocation{}}},
					{Location: &frontlinev1.KeyLocation_Header{Header: &frontlinev1.HeaderKeyLocation{Name: "X-Api-Key", StripPrefix: "Key "}}},
					{Location: &frontlinev1.KeyLocation_QueryParam{QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "api_key"}}},
				},
				Ratelimits: []*frontlinev1.KeyRatelimit{
					{Name: "tokens"},
					{Name: "burst", Limit: proto.Int64(100), Duration: proto.Int64(60000), Cost: proto.Int64(2)},
				},
			}},
		})
		require.NoError(t, err)

		require.Equal(t, "pol_KEBAP1234", got.Id)
		require.Equal(t, "keyauth", got.Name)
		require.True(t, got.Enabled)
		require.NotNil(t, got.Keyauth)
		require.Equal(t, []string{"ks_1", "ks_2"}, got.Keyauth.Keyspaces)
		require.Equal(t, ptr.P("documents.read"), got.Keyauth.PermissionQuery)

		locations := ptr.SafeDeref(got.Keyauth.Locations)
		require.Len(t, locations, 3)
		require.NotNil(t, locations[0].Bearer)
		require.Equal(t, "X-Api-Key", locations[1].Header.Name)
		require.Equal(t, ptr.P("Key "), locations[1].Header.StripPrefix)
		require.Equal(t, "api_key", locations[2].QueryParam.Name)

		ratelimits := ptr.SafeDeref(got.Keyauth.Ratelimits)
		require.Len(t, ratelimits, 2)
		require.Nil(t, ratelimits[0].Limit)
		require.Equal(t, ptr.P(int64(100)), ratelimits[1].Limit)
	})

	t.Run("keyauth without optionals omits pointers", func(t *testing.T) {
		got, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_1",
			Name:    "minimal",
			Enabled: proto.Bool(true),
			Config: &frontlinev1.Policy_Keyauth{Keyauth: &frontlinev1.KeyAuth{
				KeySpaceIds: []string{"ks_1"},
			}},
		})
		require.NoError(t, err)
		require.Nil(t, got.Match)
		require.Nil(t, got.Keyauth.Locations)
		require.Nil(t, got.Keyauth.Ratelimits)
		require.Nil(t, got.Keyauth.PermissionQuery)
	})

	t.Run("unset enabled maps to false", func(t *testing.T) {
		got, err := PolicyFromProto(&frontlinev1.Policy{
			Id:     "pol_1",
			Name:   "legacy",
			Config: &frontlinev1.Policy_Openapi{Openapi: &frontlinev1.OpenApiRequestValidation{}},
		})
		require.NoError(t, err)
		require.False(t, got.Enabled)
		require.NotNil(t, got.Openapi)
	})

	t.Run("every ratelimit identifier source", func(t *testing.T) {
		sources := []struct {
			name       string
			identifier *frontlinev1.RateLimitIdentifier
			check      func(t *testing.T, id openapi.RatelimitIdentifier)
		}{
			{
				name:       "remoteIp",
				identifier: &frontlinev1.RateLimitIdentifier{Source: &frontlinev1.RateLimitIdentifier_RemoteIp{RemoteIp: &frontlinev1.RemoteIpKey{}}},
				check:      func(t *testing.T, id openapi.RatelimitIdentifier) { require.NotNil(t, id.RemoteIp) },
			},
			{
				name:       "header",
				identifier: &frontlinev1.RateLimitIdentifier{Source: &frontlinev1.RateLimitIdentifier_Header{Header: &frontlinev1.HeaderKey{Name: "X-Tenant"}}},
				check: func(t *testing.T, id openapi.RatelimitIdentifier) {
					require.NotNil(t, id.Header)
					require.Equal(t, "X-Tenant", id.Header.Name)
				},
			},
			{
				name:       "authenticatedSubject",
				identifier: &frontlinev1.RateLimitIdentifier{Source: &frontlinev1.RateLimitIdentifier_AuthenticatedSubject{AuthenticatedSubject: &frontlinev1.AuthenticatedSubjectKey{}}},
				check: func(t *testing.T, id openapi.RatelimitIdentifier) {
					require.NotNil(t, id.AuthenticatedSubject)
				},
			},
			{
				name:       "path",
				identifier: &frontlinev1.RateLimitIdentifier{Source: &frontlinev1.RateLimitIdentifier_Path{Path: &frontlinev1.PathKey{}}},
				check:      func(t *testing.T, id openapi.RatelimitIdentifier) { require.NotNil(t, id.Path) },
			},
			{
				name:       "principalField",
				identifier: &frontlinev1.RateLimitIdentifier{Source: &frontlinev1.RateLimitIdentifier_PrincipalField{PrincipalField: &frontlinev1.PrincipalFieldKey{Path: "sub"}}},
				check: func(t *testing.T, id openapi.RatelimitIdentifier) {
					require.NotNil(t, id.PrincipalField)
					require.Equal(t, "sub", id.PrincipalField.Path)
				},
			},
		}

		for _, tc := range sources {
			t.Run(tc.name, func(t *testing.T) {
				got, err := PolicyFromProto(&frontlinev1.Policy{
					Id:      "pol_1",
					Name:    "rl",
					Enabled: proto.Bool(true),
					Config: &frontlinev1.Policy_Ratelimit{Ratelimit: &frontlinev1.RateLimit{
						Limit:      100,
						WindowMs:   60000,
						Identifier: tc.identifier,
					}},
				})
				require.NoError(t, err)
				require.NotNil(t, got.Ratelimit)
				require.Equal(t, int64(100), got.Ratelimit.Limit)
				require.Equal(t, int64(60000), got.Ratelimit.WindowMs)
				require.NotNil(t, got.Ratelimit.Identifier)
				require.Nil(t, got.Ratelimit.Identifiers)
				tc.check(t, *got.Ratelimit.Identifier)
			})
		}
	})

	t.Run("compound ratelimit identifiers render as a list", func(t *testing.T) {
		got, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_1",
			Name:    "rl",
			Enabled: proto.Bool(true),
			Config: &frontlinev1.Policy_Ratelimit{Ratelimit: &frontlinev1.RateLimit{
				Limit:    100,
				WindowMs: 60000,
				Identifiers: []*frontlinev1.RateLimitIdentifier{
					{Source: &frontlinev1.RateLimitIdentifier_AuthenticatedSubject{AuthenticatedSubject: &frontlinev1.AuthenticatedSubjectKey{}}},
					{Source: &frontlinev1.RateLimitIdentifier_Path{Path: &frontlinev1.PathKey{}}},
				},
			}},
		})
		require.NoError(t, err)
		require.NotNil(t, got.Ratelimit)
		require.Nil(t, got.Ratelimit.Identifier)
		require.NotNil(t, got.Ratelimit.Identifiers)
		identifiers := *got.Ratelimit.Identifiers
		require.Len(t, identifiers, 2)
		require.NotNil(t, identifiers[0].AuthenticatedSubject)
		require.NotNil(t, identifiers[1].Path)
	})

	t.Run("ratelimit without any identifier is unmappable", func(t *testing.T) {
		_, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_1",
			Name:    "rl",
			Enabled: proto.Bool(true),
			Config: &frontlinev1.Policy_Ratelimit{Ratelimit: &frontlinev1.RateLimit{
				Limit:    100,
				WindowMs: 60000,
			}},
		})
		require.Error(t, err)
	})

	t.Run("firewall action renders by enum name", func(t *testing.T) {
		got, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_1",
			Name:    "deny",
			Enabled: proto.Bool(false),
			Config:  &frontlinev1.Policy_Firewall{Firewall: &frontlinev1.Firewall{Action: frontlinev1.Action_ACTION_DENY}},
		})
		require.NoError(t, err)
		require.Equal(t, openapi.FirewallPolicyAction("ACTION_DENY"), got.Firewall.Action)
	})

	t.Run("jwtauth is unmappable", func(t *testing.T) {
		_, err := PolicyFromProto(&frontlinev1.Policy{
			Id:     "pol_1",
			Name:   "jwt",
			Config: &frontlinev1.Policy_Jwtauth{Jwtauth: &frontlinev1.JWTAuth{}},
		})
		require.Error(t, err)
	})

	t.Run("missing config is unmappable", func(t *testing.T) {
		_, err := PolicyFromProto(&frontlinev1.Policy{Id: "pol_1", Name: "empty"})
		require.Error(t, err)
	})

	t.Run("keyauth without keyspaces is unmappable", func(t *testing.T) {
		_, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_1",
			Name:    "no-keyspaces",
			Enabled: proto.Bool(true),
			Config:  &frontlinev1.Policy_Keyauth{Keyauth: &frontlinev1.KeyAuth{}},
		})
		require.Error(t, err)
	})

	t.Run("key location without variant is unmappable", func(t *testing.T) {
		_, err := PolicyFromProto(&frontlinev1.Policy{
			Id:      "pol_1",
			Name:    "empty-location",
			Enabled: proto.Bool(true),
			Config: &frontlinev1.Policy_Keyauth{Keyauth: &frontlinev1.KeyAuth{
				KeySpaceIds: []string{"ks_KEBAP"},
				Locations:   []*frontlinev1.KeyLocation{{}},
			}},
		})
		require.Error(t, err)
	})
}

func TestMapMatchExprFromProto(t *testing.T) {
	t.Run("path with ignoreCase", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{
					IgnoreCase: true,
					Match:      &frontlinev1.StringMatch_Prefix{Prefix: "/internal/"},
				},
			}},
		})
		require.NoError(t, err)
		require.NotNil(t, got.Path)
		require.Equal(t, ptr.P("/internal/"), got.Path.Path.Prefix)
		require.Equal(t, ptr.P(true), got.Path.Path.IgnoreCase)
		require.Nil(t, got.Path.Path.Exact)
		require.Nil(t, got.Path.Path.Regex)
	})

	t.Run("ignoreCase false is omitted", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Exact{Exact: "/health"}},
			}},
		})
		require.NoError(t, err)
		require.Nil(t, got.Path.Path.IgnoreCase)
		require.Equal(t, ptr.P("/health"), got.Path.Path.Exact)
	})

	t.Run("regex", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Regex{Regex: "^/v[0-9]+/"}},
			}},
		})
		require.NoError(t, err)
		require.Equal(t, ptr.P("^/v[0-9]+/"), got.Path.Path.Regex)
	})

	t.Run("method", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Method{Method: &frontlinev1.MethodMatch{Methods: []string{"GET", "DELETE"}}},
		})
		require.NoError(t, err)
		require.NotNil(t, got.Method)
		require.Equal(t, []openapi.MethodMatchMethods{"GET", "DELETE"}, got.Method.Methods)
	})

	t.Run("header present", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Header{Header: &frontlinev1.HeaderMatch{
				Name:  "X-Debug",
				Match: &frontlinev1.HeaderMatch_Present{Present: true},
			}},
		})
		require.NoError(t, err)
		require.NotNil(t, got.Header)
		require.Equal(t, "X-Debug", got.Header.Name)
		require.Equal(t, ptr.P(openapi.FieldMatchPresent(true)), got.Header.Present)
		require.Nil(t, got.Header.Value)
	})

	t.Run("header value", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Header{Header: &frontlinev1.HeaderMatch{
				Name:  "X-Env",
				Match: &frontlinev1.HeaderMatch_Value{Value: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Exact{Exact: "KEBAP"}}},
			}},
		})
		require.NoError(t, err)
		require.Nil(t, got.Header.Present)
		require.Equal(t, ptr.P("KEBAP"), got.Header.Value.Exact)
	})

	t.Run("queryParam value", func(t *testing.T) {
		got, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_QueryParam{QueryParam: &frontlinev1.QueryParamMatch{
				Name:  "v",
				Match: &frontlinev1.QueryParamMatch_Value{Value: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Exact{Exact: "1"}}},
			}},
		})
		require.NoError(t, err)
		require.NotNil(t, got.QueryParam)
		require.Equal(t, "v", got.QueryParam.Name)
		require.Equal(t, ptr.P("1"), got.QueryParam.Value.Exact)
	})

	t.Run("empty expr is unmappable", func(t *testing.T) {
		_, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{})
		require.Error(t, err)
	})

	t.Run("header without match is unmappable", func(t *testing.T) {
		_, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Header{Header: &frontlinev1.HeaderMatch{Name: "X-Debug"}},
		})
		require.Error(t, err)
	})

	t.Run("string match without variant is unmappable", func(t *testing.T) {
		_, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{Path: &frontlinev1.StringMatch{}}},
		})
		require.Error(t, err)
	})

	// The gateway can match on absence (present=false) but the response
	// schema only admits `present: true`, so such a policy must error
	// instead of emitting a schema-violating response.
	t.Run("header absent-match is unmappable", func(t *testing.T) {
		_, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_Header{Header: &frontlinev1.HeaderMatch{
				Name:  "X-Debug",
				Match: &frontlinev1.HeaderMatch_Present{Present: false},
			}},
		})
		require.Error(t, err)
	})

	t.Run("queryParam absent-match is unmappable", func(t *testing.T) {
		_, err := mapMatchExprFromProto(&frontlinev1.MatchExpr{
			Expr: &frontlinev1.MatchExpr_QueryParam{QueryParam: &frontlinev1.QueryParamMatch{
				Name:  "debug",
				Match: &frontlinev1.QueryParamMatch_Present{Present: false},
			}},
		})
		require.Error(t, err)
	})
}
