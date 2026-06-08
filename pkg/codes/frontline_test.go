package codes_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

// TestFrontlineURNsEncodeAttributionDomain locks in the URN grammar
// err:frontline:<domain>:<specific>. The domain segment is what the metric's
// fault_domain label and the alert selectors read, so a drift here silently
// breaks alert routing.
func TestFrontlineURNsEncodeAttributionDomain(t *testing.T) {
	tests := []struct {
		code codes.Code
		urn  codes.URN
	}{
		// upstream
		{codes.Frontline.Proxy.BadGateway, "err:frontline:upstream:bad_gateway"},
		{codes.Frontline.Proxy.ServiceUnavailable, "err:frontline:upstream:service_unavailable"},
		{codes.Frontline.Proxy.GatewayTimeout, "err:frontline:upstream:gateway_timeout"},
		{codes.Frontline.Proxy.ProxyForwardFailed, "err:frontline:upstream:proxy_forward_failed"},
		// routing
		{codes.Frontline.Routing.ConfigNotFound, "err:frontline:routing:config_not_found"},
		{codes.Frontline.Routing.DeploymentNotFound, "err:frontline:routing:deployment_not_found"},
		// capacity
		{codes.Frontline.Routing.NoRunningInstances, "err:frontline:capacity:no_running_instances"},
		// platform
		{codes.Frontline.Routing.DeploymentSelectionFailed, "err:frontline:platform:deployment_selection_failed"},
		{codes.Frontline.Internal.InternalServerError, "err:frontline:platform:internal_server_error"},
		{codes.Frontline.Internal.ConfigLoadFailed, "err:frontline:platform:config_load_failed"},
		{codes.Frontline.Internal.InvalidConfiguration, "err:frontline:config:invalid_configuration"},
		// client
		{codes.Frontline.Auth.MissingCredentials, "err:frontline:client:missing_credentials"},
		{codes.Frontline.Auth.InvalidKey, "err:frontline:client:invalid_key"},
		{codes.Frontline.Auth.InsufficientPermissions, "err:frontline:client:insufficient_permissions"},
		{codes.Frontline.Auth.RateLimited, "err:frontline:client:rate_limited"},
		{codes.Frontline.Firewall.Denied, "err:frontline:client:firewall_denied"},
		{codes.Frontline.OpenApi.InvalidRequest, "err:frontline:client:openapi_validation_failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.urn), func(t *testing.T) {
			require.Equal(t, tt.urn, tt.code.URN())
		})
	}
}
