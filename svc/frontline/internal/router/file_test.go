package router_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

func TestFileRouter_ResolvesConfiguredHostsAndPolicies(t *testing.T) {
	t.Parallel()
	path := writeRoutes(t, `
[[local-dev.routes]]
hostname = "api.localhost"
upstream = "127.0.0.1:3000"
[[local-dev.routes.policies]]
id = "keys"
enabled = true
[local-dev.routes.policies.keyauth]
key_space_ids = ["ks_primary", "ks_secondary"]
credits = 0
permission_query = "orders.read"

[[local-dev.routes.policies]]
id = "disabled"
enabled = false
firewall = {action = "ACTION_DENY"}

[[local-dev.routes]]
hostname = "grpc.localhost"
upstream = "[::1]:4000"
protocol = "h2c"
`)
	svc, err := newFile(t, path)
	require.NoError(t, err)
	var routes router.Service = svc
	decision, err := routes.Route(t.Context(), "api.localhost")
	require.NoError(t, err)
	require.Equal(t, router.DestinationLocalInstance, decision.Destination)
	require.Equal(t, db.DeploymentsUpstreamProtocolHttp1, decision.UpstreamProtocol)
	require.Len(t, decision.LocalInstances, 1)
	require.Equal(t, "127.0.0.1:3000", decision.LocalInstances[0].Address)
	require.Empty(t, decision.RemoteRegionAddress)
	require.Len(t, decision.Policies, 2)
	require.Equal(t, "keys", decision.Policies[0].GetId())
	require.True(t, decision.Policies[0].GetEnabled())
	require.Equal(t, "disabled", decision.Policies[1].GetId())
	require.False(t, decision.Policies[1].GetEnabled())
	auth := decision.Policies[0].GetKeyauth()
	require.NotNil(t, auth)
	require.Equal(t, []string{"ks_primary", "ks_secondary"}, auth.GetKeySpaceIds())
	require.NotNil(t, auth.Credits)
	require.Zero(t, *auth.Credits)
	require.Equal(t, "orders.read", auth.GetPermissionQuery())
	require.NoError(t, routes.ValidateHostname(t.Context(), "api.localhost"))

	decision, err = routes.Route(t.Context(), "grpc.localhost")
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsUpstreamProtocolH2c, decision.UpstreamProtocol)
	require.Len(t, decision.LocalInstances, 1)
	require.Equal(t, "[::1]:4000", decision.LocalInstances[0].Address)
	require.Empty(t, decision.Policies)
}

func TestFileRouter_MatchesOnlyConfiguredHostnames(t *testing.T) {
	t.Parallel()
	svc, err := newFile(t, writeRoutes(t, `
[[local-dev.routes]]
hostname = "API.Localhost."
upstream = "localhost:3000"
`))
	require.NoError(t, err)
	for _, tt := range []struct {
		host  string
		found bool
	}{
		{"api.localhost", true},
		{"API.LOCALHOST.", true},
		{"other.localhost", false},
		{"sub.api.localhost", false},
		{"api.localhost.evil", false},
		{"", false},
	} {
		t.Run(tt.host, func(t *testing.T) {
			t.Parallel()
			decision, err := svc.Route(t.Context(), tt.host)
			if tt.found {
				require.NoError(t, err)
				require.Len(t, decision.LocalInstances, 1)
				require.Equal(t, "localhost:3000", decision.LocalInstances[0].Address)
				require.NoError(t, svc.ValidateHostname(t.Context(), tt.host))
				return
			}
			require.Error(t, err)
			require.Empty(t, decision.LocalInstances)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.Routing.ConfigNotFound.URN(), code)
			require.Error(t, svc.ValidateHostname(t.Context(), tt.host))
		})
	}
}

func TestFileRouter_RejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	validRoute := "[[local-dev.routes]]\nhostname = 'api.localhost'\nupstream = 'localhost:3000'\n"
	for _, tt := range []struct{ name, content string }{
		{"no routes", ""},
		{"old dev section", "[[dev.routes]]\nhostname='api.localhost'\nupstream='localhost:3000'"},
		{"routes outside local-dev", "[[routes]]\nhostname='api.localhost'\nupstream='localhost:3000'"},
		{"duplicate hostname", validRoute + "[[local-dev.routes]]\nhostname = 'API.Localhost.'\nupstream = 'localhost:4000'"},
		{"missing hostname", "[[local-dev.routes]]\nupstream = 'localhost:3000'"},
		{"wildcard hostname", "[[local-dev.routes]]\nhostname = '*.localhost'\nupstream = 'localhost:3000'"},
		{"hostname with port", "[[local-dev.routes]]\nhostname = 'localhost:8080'\nupstream = 'localhost:3000'"},
		{"unsupported protocol", validRoute + "protocol = 'h3'"},
		{"unknown policy", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nenabled=true\nkey_auth={}"},
		{"unknown policy option", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nenabled=true\nkeyauth={keyspacez=['ks_test']}"},
		{"missing enabled", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nkeyauth={}"},
		{"missing policy config", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nenabled=true"},
		{"missing policy ID", validRoute + "[[local-dev.routes.policies]]\nenabled=true\nkeyauth={}"},
		{"duplicate policy ID", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nenabled=true\nkeyauth={}\n[[local-dev.routes.policies]]\nid='auth'\nenabled=false\nkeyauth={}"},
		{"unimplemented JWT policy", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nenabled=true\njwtauth={}"},
		{"empty match expression", validRoute + "[[local-dev.routes.policies]]\nid='auth'\nenabled=true\nkeyauth={}\nmatch=[{}]"},
		{"missing firewall action", validRoute + "[[local-dev.routes.policies]]\nid='block'\nenabled=true\nfirewall={}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, err := newFile(t, writeRoutes(t, tt.content))
			require.Error(t, err)
			require.Nil(t, svc)
		})
	}
	for _, upstream := range []string{"", "localhost", ":3000", "localhost:0", "localhost:65536", "localhost:abc", "http://localhost:3000", "user@localhost:3000", "localhost:3000/path", "localhost:3000?query", "bad host:3000"} {
		t.Run("upstream="+upstream, func(t *testing.T) {
			t.Parallel()
			svc, err := newFile(t, writeRoutes(t, fmt.Sprintf("[[local-dev.routes]]\nhostname='api.localhost'\nupstream=%q", upstream)))
			require.Error(t, err)
			require.Nil(t, svc)
		})
	}
}

func TestFileRouter_LoadsOpenAPISpecRelativeToRoutesFile(t *testing.T) {
	t.Parallel()
	path := writeRoutes(t, `
[[local-dev.routes]]
hostname = "api.localhost"
upstream = "localhost:3000"
openapi_spec = "schemas/orders.yaml"
[[local-dev.routes.policies]]
id = "validate"
enabled = true
openapi = {}
`)
	spec := []byte(`openapi: 3.0.3
info:
  title: Orders
  version: "1"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`)
	specPath := filepath.Join(filepath.Dir(path), "schemas", "orders.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(specPath), 0700))
	require.NoError(t, os.WriteFile(specPath, spec, 0600))
	svc, err := newFile(t, path)
	require.NoError(t, err)
	decision, err := svc.Route(t.Context(), "api.localhost")
	require.NoError(t, err)
	require.Len(t, decision.Policies, 1)
	require.Equal(t, spec, decision.Policies[0].GetOpenapi().GetSpecYaml())
}

func TestFileRouter_RejectsIncompleteOpenAPIConfiguration(t *testing.T) {
	t.Parallel()
	validSpec := `{"openapi":"3.0.3","info":{"title":"Orders","version":"1"},"paths":{}}`
	for _, tt := range []struct {
		name, spec, policy string
		reference, write   bool
	}{
		{name: "no spec", policy: "openapi={}"},
		{name: "missing file", reference: true, policy: "openapi={}"},
		{name: "empty file", reference: true, write: true, policy: "openapi={}"},
		{name: "invalid spec", reference: true, write: true, spec: "not an OpenAPI spec", policy: "openapi={}"},
		{name: "spec without policy", reference: true, write: true, spec: validSpec},
		{name: "inline and file conflict", reference: true, write: true, spec: validSpec, policy: "openapi={spec_yaml='e30='}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := "[[local-dev.routes]]\nhostname='api.localhost'\nupstream='localhost:3000'\n"
			if tt.reference {
				content += "openapi_spec='openapi.json'\n"
			}
			if tt.policy != "" {
				content += "[[local-dev.routes.policies]]\nid='validate'\nenabled=true\n" + tt.policy
			}
			path := writeRoutes(t, content)
			if tt.write {
				require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(path), "openapi.json"), []byte(tt.spec), 0600))
			}
			svc, err := newFile(t, path)
			require.Error(t, err)
			require.Nil(t, svc)
		})
	}
}

func newFile(t *testing.T, path string) (router.Service, error) {
	t.Helper()
	var cfg struct {
		LocalDev struct {
			Routes []router.FileRoute `toml:"routes"`
		} `toml:"local-dev"`
	}
	_, err := toml.DecodeFile(path, &cfg)
	require.NoError(t, err)
	svc, err := router.NewFile(cfg.LocalDev.Routes, filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func writeRoutes(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routes.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}
