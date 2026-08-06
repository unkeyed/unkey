package policyconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
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
			},
		},
		{
			name:     "no variant set",
			policies: []openapi.Policy{{Name: "empty", Enabled: true}},
			wantErr:  "policies[0] must set exactly one of keyauth, ratelimit, firewall or openapi; none are set.",
		},
		{
			name: "two variants set",
			policies: []openapi.Policy{{
				Name: "double", Enabled: true, Firewall: firewall,
				Openapi: &openapi.OpenapiPolicy{},
			}},
			wantErr: "policies[0] must set exactly one of keyauth, ratelimit, firewall or openapi; 2 are set.",
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
				Match: &[]openapi.MatchExpr{{Path: &openapi.PathMatch{Path: openapi.StringMatch{Exact: ptr.P("/a"), Prefix: ptr.P("/b")}}}},
			}},
			wantErr: "policies[0].match[0].path.path must set exactly one of",
		},
		{
			name: "invalid regex",
			policies: []openapi.Policy{{
				Name: "m", Enabled: true, Firewall: firewall,
				Match: &[]openapi.MatchExpr{{Path: &openapi.PathMatch{Path: openapi.StringMatch{Regex: ptr.P("[unclosed")}}}},
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
				Match: &[]openapi.MatchExpr{{Header: &openapi.FieldMatch{Name: "x-kebap", Present: &present, Value: &openapi.StringMatch{Exact: ptr.P("v")}}}},
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
					PermissionQuery: ptr.P("(documents.read OR documents.list) AND kebap.eat"),
				},
			}},
		},
		{
			name: "keyauth with malformed permission query",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces:       []string{"ks_1"},
					PermissionQuery: ptr.P("documents.read AND AND documents.write"),
				},
			}},
			wantErr: "policies[0].keyauth.permissionQuery is not a valid permission query",
		},
		{
			name: "keyauth ratelimit with limit but no duration",
			policies: []openapi.Policy{{
				Name: "k", Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces:  []string{"ks_1"},
					Ratelimits: &[]openapi.KeyRatelimit{{Name: "requests", Limit: ptr.P(int64(10))}},
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
