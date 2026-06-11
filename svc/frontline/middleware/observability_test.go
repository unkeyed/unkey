package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
)

// TestGetErrorPageInfoFrontline_StatusMapping locks in the URN → HTTP status
// mapping. A new URN that lacks an explicit case will fall through to the
// default branch and be reported as 500, which causes false 5xx alerts.
//
// When you add a new URN that frontline can produce, add a case here too.
func TestGetErrorPageInfoFrontline_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		urn        codes.URN
		wantStatus int
	}{
		// Client-side / user errors — must not be 5xx.
		{codes.User.BadRequest.ClientClosedRequest.URN(), 499},
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

		// Deployment's own config is invalid — caller-side (422), not a 5xx
		// frontline fault.
		{codes.Frontline.Internal.InvalidConfiguration.URN(), http.StatusUnprocessableEntity},

		// Genuine server-side faults — 5xx is correct.
		{codes.Frontline.Routing.DeploymentSelectionFailed.URN(), http.StatusInternalServerError},
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

type stubRenderer struct{}

func (stubRenderer) Render(errorpage.Data) ([]byte, error) {
	return []byte("<html>error</html>"), nil
}

func TestWithObservability_JSONErrorIncludesMetaRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		urn    codes.URN
		accept string
	}{
		{
			name:   "application/json",
			urn:    codes.Frontline.Proxy.GatewayTimeout.URN(),
			accept: "application/json",
		},
		{
			name:   "application/* without text/html",
			urn:    codes.Frontline.Proxy.ServiceUnavailable.URN(),
			accept: "application/*",
		},
		{
			name:   "*/* without text/html",
			urn:    codes.Frontline.Internal.InternalServerError.URN(),
			accept: "*/*",
		},
	}

	mw := WithObservability(stubRenderer{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", tt.accept)

			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, zen.SessionConfig{}))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("upstream failed", fault.Code(tt.urn))
			})

			err := handler(context.Background(), sess)
			require.NoError(t, err)

			var body ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

			require.NotEmpty(t, body.Meta.RequestID, "JSON error must include meta.requestId")
			require.Equal(t, sess.RequestID(), body.Meta.RequestID)
			require.NotEmpty(t, body.Error.Code)
		})
	}
}

func TestWithObservability_HTMLFallbackIncludesMetaRequestID(t *testing.T) {
	t.Parallel()

	// When the HTML renderer fails the middleware falls back to JSON.
	// That fallback must also carry meta.requestId.
	mw := WithObservability(renderFunc(func(errorpage.Data) ([]byte, error) {
		return nil, fault.New("template broken")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")

	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, zen.SessionConfig{}))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Proxy.BadGateway.URN()))
	})

	err := handler(context.Background(), sess)
	require.NoError(t, err)

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	require.NotEmpty(t, body.Meta.RequestID, "JSON fallback must include meta.requestId")
	require.Equal(t, sess.RequestID(), body.Meta.RequestID)
}

// TestWithObservability_ResponseExposesURN verifies that callers receive the
// full fault URN so they can quote it to support. The same URN is logged under
// the request ID for correlation.
func TestWithObservability_ResponseExposesURN(t *testing.T) {
	t.Parallel()

	urns := []codes.URN{
		codes.Frontline.Proxy.GatewayTimeout.URN(),
		codes.Frontline.Auth.RateLimited.URN(),
		codes.Frontline.Routing.NoRunningInstances.URN(),
		codes.Frontline.Internal.InvalidConfiguration.URN(),
		codes.Frontline.Internal.InternalServerError.URN(),
	}

	mw := WithObservability(stubRenderer{})

	for _, urn := range urns {
		t.Run(string(urn), func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", "application/json")

			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, zen.SessionConfig{}))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom", fault.Code(urn))
			})
			require.NoError(t, handler(context.Background(), sess))

			var body ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

			require.Equal(t, string(urn), body.Error.Code, "caller should receive the full URN")
		})
	}
}

type renderFunc func(errorpage.Data) ([]byte, error)

func (f renderFunc) Render(d errorpage.Data) ([]byte, error) { return f(d) }
