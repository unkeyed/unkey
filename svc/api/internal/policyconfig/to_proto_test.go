package policyconfig

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestMapPoliciesToProtoValidation(t *testing.T) {
	firewall := &openapi.FirewallPolicy{Action: "ACTION_DENY"}
	present := openapi.FieldMatchPresent(true)

	testCases := []struct {
		name     string
		policies []openapi.Policy
		wantErr  string
	}{
		{
			name: "valid one of each variant",
			policies: []openapi.Policy{
				{Name: "kebap-keyauth", Enabled: true, Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{"ks_1"}}},
				{Name: "ratelimit", Enabled: true, Ratelimit: &openapi.RatelimitPolicy{
					Limit: 10, WindowMs: 1000,
					Identifier: &openapi.RatelimitIdentifier{RemoteIp: &openapi.RemoteIpKey{}},
				}},
				{Name: "firewall", Enabled: false, Firewall: firewall},
				{Name: "openapi", Enabled: true, Openapi: &openapi.OpenapiPolicy{}},
				{Name: "logging", Enabled: true, Logging: &openapi.LoggingPolicy{}},
			},
		},
		{
			name:     "no variant set",
			policies: []openapi.Policy{{Name: "empty", Enabled: true}},
			wantErr:  "policies[0] must set exactly one of keyauth, ratelimit, firewall, openapi or logging; none are set.",
		},
		{
			name: "two variants set",
			policies: []openapi.Policy{{
				Name: "double", Enabled: true, Firewall: firewall,
				Openapi: &openapi.OpenapiPolicy{},
			}},
			wantErr: "policies[0] must set exactly one of keyauth, ratelimit, firewall, openapi or logging; 2 are set.",
		},
		{
			name: "match expr with no variant",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{}},
			}},
			wantErr: "policies[0].match[0] must set exactly one of",
		},
		{
			name: "string match with two modes",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{Path: &openapi.PathMatch{Path: openapi.StringMatch{Exact: new("/a"), Prefix: new("/b")}}}},
			}},
			wantErr: "policies[0].match[0].path.path must set exactly one of",
		},
		{
			name: "invalid regex",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{Path: &openapi.PathMatch{Path: openapi.StringMatch{Regex: new("[unclosed")}}}},
			}},
			wantErr: "policies[0].match[0].path.path.regex is not a valid regular expression",
		},
		{
			name: "header match with neither present nor value",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{Header: &openapi.FieldMatch{Name: "x-kebap"}}},
			}},
			wantErr: "policies[0].match[0].header must set exactly one of present or value",
		},
		{
			name: "header match with both present and value",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{Header: &openapi.FieldMatch{Name: "x-kebap", Present: &present, Value: &openapi.StringMatch{Exact: new("v")}}}},
			}},
			wantErr: "policies[0].match[0].header must set exactly one of present or value",
		},
		{
			name: "query param match valid with present",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{QueryParam: &openapi.FieldMatch{Name: "token", Present: &present}}},
			}},
		},
		{
			name: "remote ip match with neither in nor notIn",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{}}},
			}},
			wantErr: "policies[0].match[0].remoteIp must set exactly one of in or notIn; none are set.",
		},
		{
			name: "remote ip match with both in and notIn",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{
					In:    &[]string{"203.0.113.0/24"},
					NotIn: &[]string{"198.51.100.0/24"},
				}}},
			}},
			wantErr: "policies[0].match[0].remoteIp must set exactly one of in or notIn; 2 are set.",
		},
		{
			name: "remote ip match with invalid entry",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{In: &[]string{"203.0.113.0/24", "kebap"}}}},
			}},
			wantErr: "policies[0].match[0].remoteIp.in[1] is not a valid IP or CIDR",
		},
		{
			name: "remote ip match with host bits set",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{NotIn: &[]string{"10.1.2.3/8"}}}},
			}},
			wantErr: "policies[0].match[0].remoteIp.notIn[0] has host bits set; use 10.0.0.0/8",
		},
		{
			name: "remote ip match with ipv4-mapped ipv6 entry",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{In: &[]string{"::ffff:203.0.113.7"}}}},
			}},
			wantErr: "policies[0].match[0].remoteIp.in[0] is an IPv4-mapped IPv6 address; use the IPv4 form",
		},
		{
			name: "remote ip match with zoned entry",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{In: &[]string{"fe80::1%eth0"}}}},
			}},
			wantErr: "policies[0].match[0].remoteIp.in[0] is not a valid IP or CIDR",
		},
		{
			name: "remote ip match with too many entries",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{In: new(slices.Repeat([]string{"203.0.113.0/24"}, 101))}}},
			}},
			wantErr: "policies[0].match[0].remoteIp.in must not have more than 100 entries.",
		},
		{
			name: "key location with no variant",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces: []string{"ks_1"},
					Locations: &[]openapi.KeyLocation{{}},
				},
			}},
			wantErr: "policies[0].keyauth.locations[0] must set exactly one of",
		},
		{
			name: "keyauth with valid permission query",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces:       []string{"ks_1"},
					PermissionQuery: new("(documents.read OR documents.list) AND kebap.eat"),
				},
			}},
		},
		{
			name: "keyauth with malformed permission query",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces:       []string{"ks_1"},
					PermissionQuery: new("documents.read AND AND documents.write"),
				},
			}},
			wantErr: "policies[0].keyauth.permissionQuery is not a valid permission query",
		},
		{
			name: "keyauth with valid credits override",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces: []string{"ks_1"},
					Credits:   new(int64(0)),
				},
			}},
		},
		{
			name: "keyauth with negative credits override",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces: []string{"ks_1"},
					Credits:   new(int64(-1)),
				},
			}},
			wantErr: "policies[0].keyauth.credits must not be negative",
		},
		{
			name: "keyauth ratelimit with limit but no duration",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces:  []string{"ks_1"},
					Ratelimits: &[]openapi.KeyRatelimit{{Name: "requests", Limit: new(int64(10))}},
				},
			}},
			wantErr: "policies[0].keyauth.ratelimits[0] must set limit and duration together",
		},
		{
			name: "ratelimit identifier with two variants",
			policies: []openapi.Policy{{
				Name: "r", Enabled: true,
				Ratelimit: &openapi.RatelimitPolicy{
					Limit: 10, WindowMs: 1000,
					Identifier: &openapi.RatelimitIdentifier{
						RemoteIp: &openapi.RemoteIpKey{},
						Path:     &openapi.PathKey{},
					},
				},
			}},
			wantErr: "policies[0].ratelimit.identifier must set exactly one of",
		},
		{
			name: "ratelimit with neither identifier nor identifiers",
			policies: []openapi.Policy{{
				Name: "r", Enabled: true,
				Ratelimit: &openapi.RatelimitPolicy{Limit: 10, WindowMs: 1000},
			}},
			wantErr: "policies[0].ratelimit must set exactly one of identifier or identifiers",
		},
		{
			name: "ratelimit with both identifier and identifiers",
			policies: []openapi.Policy{{
				Name: "r", Enabled: true,
				Ratelimit: &openapi.RatelimitPolicy{
					Limit: 10, WindowMs: 1000,
					Identifier: &openapi.RatelimitIdentifier{RemoteIp: &openapi.RemoteIpKey{}},
					Identifiers: &[]openapi.RatelimitIdentifier{
						{Path: &openapi.PathKey{}},
					},
				},
			}},
			wantErr: "policies[0].ratelimit must set exactly one of identifier or identifiers",
		},
		{
			name: "compound identifier entry with two variants",
			policies: []openapi.Policy{{
				Name: "r", Enabled: true,
				Ratelimit: &openapi.RatelimitPolicy{
					Limit: 10, WindowMs: 1000,
					Identifiers: &[]openapi.RatelimitIdentifier{
						{AuthenticatedSubject: &openapi.AuthenticatedSubjectKey{}},
						{RemoteIp: &openapi.RemoteIpKey{}, Path: &openapi.PathKey{}},
					},
				},
			}},
			wantErr: "policies[0].ratelimit.identifiers[1] must set exactly one of",
		},
		{
			name: "error names the failing index",
			policies: []openapi.Policy{
				{Name: "ok", Enabled: true, Firewall: firewall},
				{Name: "bad", Enabled: true},
			},
			wantErr: "policies[1] must set exactly one of",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ToProto(tc.policies)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, fault.UserFacingMessage(err), tc.wantErr)
		})
	}
}

// The deprecated single identifier input must normalize to a one-entry
// repeated identifiers proto field, so no write path populates the
// deprecated proto field anymore.
func TestLegacyIdentifierNormalizesToRepeated(t *testing.T) {
	got, err := ToProto([]openapi.Policy{{
		Name: "rl", Enabled: true,
		Ratelimit: &openapi.RatelimitPolicy{
			Limit: 10, WindowMs: 1000,
			Identifier: &openapi.RatelimitIdentifier{RemoteIp: &openapi.RemoteIpKey{}},
		},
	}})
	require.NoError(t, err)
	require.Len(t, got, 1)

	ratelimit := got[0].GetRatelimit()
	require.NotNil(t, ratelimit)
	require.Nil(t, ratelimit.GetIdentifier())
	require.Len(t, ratelimit.GetIdentifiers(), 1)
	require.NotNil(t, ratelimit.GetIdentifiers()[0].GetRemoteIp())
}

func TestRemoteIpMatchToProtoNormalizesEntries(t *testing.T) {
	policy, err := PolicyToProto("policies[0]", openapi.Policy{
		Name: "office only", Enabled: true,
		Firewall: &openapi.FirewallPolicy{Action: "ACTION_DENY"},
		Match: &[]openapi.MatchExpr{{RemoteIp: &openapi.RemoteIpMatch{
			NotIn: &[]string{"198.51.100.0/24", "203.0.113.7", "2001:db8::1"},
		}}},
	})
	require.NoError(t, err)
	require.Equal(t,
		[]string{"198.51.100.0/24", "203.0.113.7/32", "2001:db8::1/128"},
		policy.GetMatch()[0].GetRemoteIp().GetNotIn(),
	)

	back, err := PolicyFromProto(policy)
	require.NoError(t, err)
	require.Equal(t, policy.GetMatch()[0].GetRemoteIp().GetNotIn(), *(*back.Match)[0].RemoteIp.NotIn)
	require.Nil(t, (*back.Match)[0].RemoteIp.In)
}
