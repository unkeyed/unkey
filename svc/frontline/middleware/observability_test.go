package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

// TestGetErrorPageInfoFrontline_StatusMapping locks in the URN → HTTP
// status mapping. Most rows resolve via pkg/codes (Code.HTTPStatus +
// Category.HTTPStatus); a few resolve via frontlineStatusOverrides.
//
// A new URN that ends up returning 500 unintentionally — because either
// pkg/codes or this package forgot to map it — will fail this test.
func TestGetErrorPageInfoFrontline_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		urn        codes.URN
		wantStatus int
	}{
		// Client-side / user errors — must not be 5xx.
		{codes.User.BadRequest.ClientClosedRequest.URN(), 499},
		// RequestTimeout is overridden to 504 in frontline (gateway semantics)
		// even though pkg/codes maps it to 408.
		{codes.User.BadRequest.RequestTimeout.URN(), http.StatusGatewayTimeout},
		{codes.User.BadRequest.RequestBodyTooLarge.URN(), http.StatusRequestEntityTooLarge},
		{codes.User.BadRequest.RequestBodyUnreadable.URN(), http.StatusBadRequest},
		{codes.Auth.Authentication.Missing.URN(), http.StatusUnauthorized},
		{codes.Auth.Authentication.Malformed.URN(), http.StatusUnauthorized},
		{codes.App.Validation.InvalidInput.URN(), http.StatusBadRequest},

		// Frontline-specific errors with established mappings.
		{codes.Frontline.Routing.ConfigNotFound.URN(), http.StatusNotFound},
		{codes.Frontline.Routing.DeploymentNotFound.URN(), http.StatusNotFound},
		{codes.Frontline.Routing.NoRunningInstances.URN(), http.StatusServiceUnavailable},
		{codes.Frontline.Proxy.BadGateway.URN(), http.StatusBadGateway},
		{codes.Frontline.Proxy.ProxyForwardFailed.URN(), http.StatusBadGateway},
		{codes.Frontline.Proxy.ServiceUnavailable.URN(), http.StatusServiceUnavailable},
		{codes.Frontline.Proxy.GatewayTimeout.URN(), http.StatusGatewayTimeout},
		{codes.Frontline.Auth.MissingCredentials.URN(), http.StatusUnauthorized},
		{codes.Frontline.Auth.InvalidKey.URN(), http.StatusUnauthorized},
		{codes.Frontline.Auth.InsufficientPermissions.URN(), http.StatusForbidden},
		{codes.Frontline.Auth.RateLimited.URN(), http.StatusTooManyRequests},
		{codes.Frontline.Firewall.Denied.URN(), http.StatusForbidden},
		{codes.Frontline.OpenApi.InvalidRequest.URN(), http.StatusBadRequest},

		// Genuine server-side faults — 5xx is correct.
		{codes.Frontline.Routing.DeploymentSelectionFailed.URN(), http.StatusInternalServerError},
		{codes.Frontline.Internal.InvalidConfiguration.URN(), http.StatusInternalServerError},
		{codes.Frontline.Internal.InternalServerError.URN(), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.urn), func(t *testing.T) {
			t.Parallel()
			got := getErrorPageInfoFrontline(tt.urn)
			require.Equal(t, tt.wantStatus, got.Status,
				"URN %s should map to HTTP %d, got %d",
				tt.urn, tt.wantStatus, got.Status)
		})
	}
}

func TestGetErrorPageInfoFrontline_UnknownURNDefaultsTo500(t *testing.T) {
	t.Parallel()

	got := getErrorPageInfoFrontline("err:made:up:nonexistent")
	require.Equal(t, http.StatusInternalServerError, got.Status,
		"unknown URN should default to 500 so we get alerted on it")
}

func TestGetErrorPageInfoFrontline_TitleFor499(t *testing.T) {
	t.Parallel()

	// 499 is non-stdlib; status.Text() returns "". errorTitle() must
	// special-case it so the rendered page has a title.
	got := getErrorPageInfoFrontline(codes.User.BadRequest.ClientClosedRequest.URN())
	require.Equal(t, "Client Closed Request", got.Title)
}
