package policies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	firewallExec "github.com/unkeyed/unkey/svc/frontline/internal/policies/firewall"
	openapiExec "github.com/unkeyed/unkey/svc/frontline/internal/policies/openapi"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
	"google.golang.org/protobuf/encoding/protojson"
)

type keyAuthenticatorFunc func(context.Context, *zen.Session, *http.Request, string, *frontlinev1.KeyAuth) (*principal.Principal, error)

func (f keyAuthenticatorFunc) Execute(ctx context.Context, sess *zen.Session, req *http.Request, appID string, cfg *frontlinev1.KeyAuth) (*principal.Principal, error) {
	return f(ctx, sess, req, appID, cfg)
}

func TestEvaluate_UsesKeyAuthenticator(t *testing.T) {
	t.Parallel()

	denied := fault.New("denied", fault.Code(codes.Frontline.Auth.InsufficientPermissions.URN()))
	for _, tt := range []struct {
		name      string
		principal *principal.Principal
		err       error
		code      codes.URN
	}{
		{name: "success", principal: &principal.Principal{Subject: "customer_42"}},
		{name: "authentication error", err: denied, code: codes.Frontline.Auth.InsufficientPermissions.URN()},
		{name: "missing principal", code: codes.Frontline.Internal.InternalServerError.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))
			cfg := &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_orders"}, Credits: new(int64(0))}
			engine := &Engine{
				keyAuth: keyAuthenticatorFunc(func(ctx context.Context, session *zen.Session, request *http.Request, appID string, policy *frontlinev1.KeyAuth) (*principal.Principal, error) {
					require.Equal(t, t.Context(), ctx)
					require.Same(t, sess, session)
					require.Same(t, req, request)
					require.Equal(t, "app_orders", appID)
					require.Same(t, cfg, policy)
					session.ResponseWriter().Header().Set("X-RateLimit-Remaining", "7")
					return tt.principal, tt.err
				}),
			}

			result, err := engine.Evaluate(t.Context(), sess, req, "ws_orders", "app_orders", []*frontlinev1.Policy{{
				Enabled: new(true),
				Config:  &frontlinev1.Policy_Keyauth{Keyauth: cfg},
			}})
			require.Equal(t, tt.principal, result.Principal)
			require.Equal(t, "7", w.Header().Get("X-RateLimit-Remaining"))
			if tt.code == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tt.code, code)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			}
		})
	}
}

func TestParseMiddleware_Nil(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware(nil)
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_Empty(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware([]byte{})
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_EmptyJSON(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware([]byte("{}"))
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_InvalidProto(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware([]byte("not a valid protobuf"))
	require.Error(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_NoPolicies(t *testing.T) {
	t.Parallel()
	mw := &frontlinev1.Config{Policies: nil}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)

	policies, err := ParseMiddleware(raw)
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_WithPolicies(t *testing.T) {
	t.Parallel()
	mw := &frontlinev1.Config{
		Policies: []*frontlinev1.Policy{
			{
				Id:      "p1",
				Name:    "key auth",
				Enabled: new(true),
				Match:   nil,
				Config: &frontlinev1.Policy_Keyauth{
					//nolint:exhaustruct
					Keyauth: &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_123"}},
				},
			},
		},
	}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)

	policies, err := ParseMiddleware(raw)
	require.NoError(t, err)
	require.NotNil(t, policies)
	require.Len(t, policies, 1)
	require.Equal(t, "p1", policies[0].GetId())
}

// TestPrincipal_Marshal_WireFormat pins the JSON wire format of the
// Principal. The header contract documented in
// docs/compute/gateway/principal.mdx is exactly this
// output — if this test changes, update the docs in the same commit.
func TestPrincipal_Marshal_WireFormat(t *testing.T) {
	t.Parallel()

	t.Run("minimal principal omits optional fields", func(t *testing.T) {
		t.Parallel()
		p := &principal.Principal{
			Version: principal.PrincipalVersion,
			Subject: "key_abc",
			Type:    principal.PrincipalTypeAPIKey,
			Source: principal.Source{
				Key: &principal.KeySource{
					KeyID:       "key_abc",
					KeySpaceID:  "ks_456",
					Name:        nil,
					ExpiresAt:   nil,
					Meta:        map[string]any{},
					Roles:       nil,
					Permissions: nil,
				},
				JWT: nil,
			},
		}

		s, err := p.Marshal()
		require.NoError(t, err)
		require.JSONEq(t, `{
			"version": "v1",
			"subject": "key_abc",
			"type": "API_KEY",
			"source": {"key": {
				"keyId": "key_abc",
				"keySpaceId": "ks_456",
				"meta": {}
			}}
		}`, s)
	})

	t.Run("populated principal includes identity and optional key fields", func(t *testing.T) {
		t.Parallel()
		p := &principal.Principal{
			Version: principal.PrincipalVersion,
			Subject: "user_42",
			Type:    principal.PrincipalTypeAPIKey,
			Identity: &principal.Identity{
				ExternalID: "user_42",
				Meta:       map[string]any{"plan": "pro"},
			},
			Source: principal.Source{
				Key: &principal.KeySource{
					KeyID:       "key_abc",
					KeySpaceID:  "ks_456",
					Name:        new("prod"),
					ExpiresAt:   new(int64(1717200000000)),
					Credits:     new(int64(42)),
					Meta:        map[string]any{},
					Roles:       []string{"admin"},
					Permissions: []string{"api.read", "api.write"},
				},
				JWT: nil,
			},
		}

		s, err := p.Marshal()
		require.NoError(t, err)
		require.JSONEq(t, `{
			"version": "v1",
			"subject": "user_42",
			"type": "API_KEY",
			"identity": {"externalId": "user_42", "meta": {"plan": "pro"}},
			"source": {"key": {
				"keyId": "key_abc",
				"keySpaceId": "ks_456",
				"name": "prod",
				"expiresAt": 1717200000000,
				"credits": 42,
				"meta": {},
				"roles": ["admin"],
				"permissions": ["api.read", "api.write"]
			}}
		}`, s)
	})
}

func TestEvaluate_FirewallDeniesOutsideRemoteIpRange(t *testing.T) {
	t.Parallel()

	policies := []*frontlinev1.Policy{{
		Id:      uid.New(uid.PolicyPrefix),
		Enabled: new(true),
		Match: []*frontlinev1.MatchExpr{
			{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Prefix{Prefix: "/admin"}},
			}}},
			{Expr: &frontlinev1.MatchExpr_RemoteIp{RemoteIp: &frontlinev1.RemoteIpMatch{NotIn: []string{"198.51.100.0/24"}}}},
		},
		Config: &frontlinev1.Policy_Firewall{Firewall: &frontlinev1.Firewall{Action: frontlinev1.Action_ACTION_DENY}},
	}}

	for _, tt := range []struct {
		name       string
		remoteAddr string
		path       string
		denied     bool
	}{
		{name: "office on admin", remoteAddr: "198.51.100.10:4321", path: "/admin/users", denied: false},
		{name: "outside on admin", remoteAddr: "203.0.113.7:4321", path: "/admin/users", denied: true},
		{name: "outside on other path", remoteAddr: "203.0.113.7:4321", path: "/users", denied: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			engine := &Engine{firewall: firewallExec.New(), regexCache: newRegexCache()}

			_, err := engine.Evaluate(t.Context(), sess, req, uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), policies)
			if !tt.denied {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.Firewall.Denied.URN(), code)
		})
	}
}

func TestEvaluate_KeyAuthSkipCannotBypassStricterPolicyThroughPathVariant(t *testing.T) {
	t.Parallel()

	denied := fault.New("denied", fault.Code(codes.Frontline.Auth.InsufficientPermissions.URN()))
	publicCfg := &frontlinev1.KeyAuth{KeySpaceIds: []string{uid.New(uid.KeySpacePrefix)}}
	adminCfg := &frontlinev1.KeyAuth{KeySpaceIds: []string{uid.New(uid.KeySpacePrefix)}}
	engine := &Engine{
		regexCache: newRegexCache(),
		keyAuth: keyAuthenticatorFunc(func(_ context.Context, _ *zen.Session, _ *http.Request, _ string, cfg *frontlinev1.KeyAuth) (*principal.Principal, error) {
			if cfg == adminCfg {
				return nil, denied
			}
			return &principal.Principal{Subject: "public_caller"}, nil
		}),
	}
	policies := []*frontlinev1.Policy{
		{
			Id:      uid.New(uid.PolicyPrefix),
			Enabled: new(true),
			Match: []*frontlinev1.MatchExpr{{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Prefix{Prefix: "/public/"}},
			}}}},
			Config: &frontlinev1.Policy_Keyauth{Keyauth: publicCfg},
		},
		{
			Id:      uid.New(uid.PolicyPrefix),
			Enabled: new(true),
			Match: []*frontlinev1.MatchExpr{{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Exact{Exact: "/admin"}},
			}}}},
			Config: &frontlinev1.Policy_Keyauth{Keyauth: adminCfg},
		},
	}

	for _, target := range []string{"/public/..;/admin", "/public/%2e%2e/admin", "/public/x/%2e%2e/%2e%2e/admin"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, target, nil)
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

			_, err := engine.Evaluate(t.Context(), sess, req, uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), policies)
			require.ErrorIs(t, err, denied)
		})
	}
}

func TestEvaluate_RejectsEncodedPathSeparator(t *testing.T) {
	t.Parallel()

	policies := []*frontlinev1.Policy{{
		Id:      uid.New(uid.PolicyPrefix),
		Enabled: new(true),
		Match: []*frontlinev1.MatchExpr{{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
			Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Prefix{Prefix: "/internal/"}},
		}}}},
		Config: &frontlinev1.Policy_Firewall{Firewall: &frontlinev1.Firewall{Action: frontlinev1.Action_ACTION_DENY}},
	}}

	for _, tt := range []struct {
		target   string
		policies []*frontlinev1.Policy
		rejected bool
	}{
		{target: "/files/a%2Fb", policies: policies, rejected: true},
		{target: "/files/a%2fb", policies: policies, rejected: true},
		{target: "/public/x%2F..%2F..%2Finternal/x", policies: policies, rejected: true},
		{target: "/files/a%5Cb", policies: policies, rejected: true},
		{target: "/files/a%20b", policies: policies, rejected: false},
		{target: "/files/a%2Fb", policies: nil, rejected: false},
	} {
		t.Run(tt.target, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			engine := &Engine{firewall: firewallExec.New(), regexCache: newRegexCache()}

			_, err := engine.Evaluate(t.Context(), sess, req, uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), tt.policies)
			if !tt.rejected {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.Request.EncodedPathSeparator.URN(), code)
		})
	}
}

func TestEvaluate_OpenAPIValidatesPathAndMethodVariants(t *testing.T) {
	t.Parallel()

	spec := []byte(`openapi: 3.0.0
info: {title: Users, version: "1"}
paths:
  /users:
    get:
      responses:
        "200": {description: ok}
`)
	policies := []*frontlinev1.Policy{{
		Id:      uid.New(uid.PolicyPrefix),
		Enabled: new(true),
		Match: []*frontlinev1.MatchExpr{
			{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Exact{Exact: "/users"}},
			}}},
			{Expr: &frontlinev1.MatchExpr_Method{Method: &frontlinev1.MethodMatch{Methods: []string{http.MethodGet}}}},
		},
		Config: &frontlinev1.Policy_Openapi{Openapi: &frontlinev1.OpenApiRequestValidation{SpecYaml: spec}},
	}}

	for _, tt := range []struct {
		name     string
		method   string
		target   string
		rejected bool
	}{
		{name: "spec path", method: http.MethodGet, target: "/users", rejected: false},
		{name: "head on a get path passes validation", method: http.MethodHead, target: "/users", rejected: false},
		{name: "trailing slash passes validation", method: http.MethodGet, target: "/users/", rejected: false},
		{name: "case variant is validated and rejected", method: http.MethodGet, target: "/Users", rejected: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			openapi, err := openapiExec.New(clock.New(), 0, 0)
			require.NoError(t, err)
			t.Cleanup(openapi.Close)
			engine := &Engine{openapi: openapi, regexCache: newRegexCache()}

			req := httptest.NewRequest(tt.method, tt.target, nil)
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

			_, err = engine.Evaluate(t.Context(), sess, req, uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), policies)
			if !tt.rejected {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.OpenApi.InvalidRequest.URN(), code)
		})
	}
}
