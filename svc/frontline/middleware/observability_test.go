package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
)

type stubRenderer struct {
	gotData errorpage.Data
}

func (r *stubRenderer) Render(d errorpage.Data) ([]byte, error) {
	r.gotData = d
	return []byte("<html>error</html>"), nil
}

// TestWithObservability_JSONResponseUsesPublicCode asserts the JSON
// envelope returned to API callers carries the public problem code,
// not the internal URN. This is the contract that prevents leaking
// frontline-internal topology (peer_frontline_*, upstream_*, etc.) to
// customers.
func TestWithObservability_JSONResponseUsesPublicCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		urn        codes.URN
		wantStatus int
		wantCode   string
	}{
		{
			name:       "peer-frontline failure collapses to service_unavailable",
			urn:        codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "service_unavailable",
		},
		{
			name:       "upstream timeout collapses to gateway_timeout",
			urn:        codes.Frontline.Proxy.DialTimeout.URN(),
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   "gateway_timeout",
		},
		{
			name:       "config load failure collapses to internal_server_error",
			urn:        codes.Frontline.Internal.ConfigLoadFailed.URN(),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_server_error",
		},
		{
			name:       "invalid key passes through specific",
			urn:        codes.Frontline.Auth.InvalidKey.URN(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "invalid_key",
		},
		{
			name:       "rate limited passes through specific",
			urn:        codes.Frontline.Auth.RateLimited.URN(),
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "rate_limited",
		},
		{
			name:       "firewall denial passes through specific",
			urn:        codes.Frontline.Firewall.Denied.URN(),
			wantStatus: http.StatusForbidden,
			wantCode:   "firewall_denied",
		},
	}

	mw := WithObservability(&stubRenderer{})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom", fault.Code(tc.urn))
			})

			require.NoError(t, handler(context.Background(), sess))
			require.Equal(t, tc.wantStatus, w.Code)

			var body ProblemResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

			require.Equal(t, tc.wantCode, body.Code,
				"JSON response must expose public code, not internal URN")
			require.NotContains(t, body.Code, ":",
				"public code must not be a URN (no colons)")
			require.Equal(t, instanceURN(sess.RequestID()), body.Instance,
				"instance must be the request URN (RFC 9457 §3.1.5)")
			require.Equal(t, sess.RequestID(), w.Header().Get("X-Unkey-Request-Id"),
				"X-Unkey-Request-Id header carries the bare request ID")
			require.Equal(t, tc.wantStatus, body.Status,
				"status field must match HTTP status")
		})
	}
}

// TestWithObservability_HTMLPageUsesPublicCode asserts the HTML error
// page renderer receives the public code and public docs URL, not the
// internal URN.
func TestWithObservability_HTMLPageUsesPublicCode(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderer{}
	mw := WithObservability(renderer)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom",
			fault.Code(codes.Frontline.Proxy.PeerFrontlineDNSTimeout.URN()))
	})
	require.NoError(t, handler(context.Background(), sess))

	require.Equal(t, "gateway_timeout", renderer.gotData.ErrorCode,
		"HTML page must show public code, not internal URN")
	require.NotContains(t, renderer.gotData.ErrorCode, ":",
		"public code must not be a URN")
	require.NotContains(t, renderer.gotData.ErrorCode, "peer_frontline",
		"HTML page must not leak peer-frontline topology")
	require.NotContains(t, renderer.gotData.DocsURL, "peer_frontline",
		"docs URL must not leak peer-frontline topology")
	require.True(t, strings.HasPrefix(renderer.gotData.DocsURL, "https://"),
		"docs URL must be an absolute https URL")
}

// TestWithObservability_ProblemJSONContentType verifies clients that
// send Accept: application/problem+json receive the matching content
// type (RFC 9457 §6) while application/json clients receive plain
// application/json. The body is identical in both cases.
func TestWithObservability_ProblemJSONContentType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		accept          string
		wantContentType string
	}{
		{"problem+json", "application/problem+json", "application/problem+json"},
		{"plain json", "application/json", "application/json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mw := WithObservability(&stubRenderer{})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", tc.accept)
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom",
					fault.Code(codes.Frontline.Auth.InvalidKey.URN()))
			})
			require.NoError(t, handler(context.Background(), sess))

			require.Equal(t, tc.wantContentType, w.Header().Get("Content-Type"))
		})
	}
}

// TestWithObservability_RetryHints verifies retry backoff is carried by
// the standard HTTP Retry-After header (set when the catalog has a
// default) and is NOT leaked into the response body — the body is not
// machine-consumable by spec-generated SDKs, so retry hints live on the
// header only. See the ProblemResponse doc.
func TestWithObservability_RetryHints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		urn               codes.URN
		wantRetryAfterHdr bool // HTTP header set
	}{
		{
			name:              "rate_limited sets backoff header",
			urn:               codes.Frontline.Auth.RateLimited.URN(),
			wantRetryAfterHdr: true,
		},
		{
			name:              "service_unavailable sets backoff header",
			urn:               codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
			wantRetryAfterHdr: true,
		},
		{
			name:              "invalid_key has no backoff header",
			urn:               codes.Frontline.Auth.InvalidKey.URN(),
			wantRetryAfterHdr: false,
		},
		{
			name:              "firewall_denied has no backoff header",
			urn:               codes.Frontline.Firewall.Denied.URN(),
			wantRetryAfterHdr: false,
		},
		{
			name:              "config_not_found has no backoff header",
			urn:               codes.Frontline.Routing.ConfigNotFoundForCustomDomain.URN(),
			wantRetryAfterHdr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mw := WithObservability(&stubRenderer{})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom", fault.Code(tc.urn))
			})
			require.NoError(t, handler(context.Background(), sess))

			// The body must never carry retry hints.
			var raw map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
			require.NotContains(t, raw, "retryable", "body must not contain retryable")
			require.NotContains(t, raw, "retry_after", "body must not contain retry_after")

			if tc.wantRetryAfterHdr {
				require.NotEmpty(t, w.Header().Get("Retry-After"),
					"HTTP Retry-After header must be set when retry_after is in catalog")
			} else {
				require.Empty(t, w.Header().Get("Retry-After"),
					"HTTP Retry-After header must be absent when catalog has no default")
			}
		})
	}
}

func TestWithObservability_JSONErrorIncludesInstance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		urn    codes.URN
		accept string
	}{
		{
			name:   "application/json",
			urn:    codes.Frontline.Proxy.UpstreamResponseTimeout.URN(),
			accept: "application/json",
		},
		{
			name:   "application/problem+json",
			urn:    codes.Frontline.Proxy.UpstreamConnectionRefused.URN(),
			accept: "application/problem+json",
		},
		{
			name:   "*/* without text/html",
			urn:    codes.Frontline.Internal.InternalServerError.URN(),
			accept: "*/*",
		},
	}

	mw := WithObservability(&stubRenderer{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept", tt.accept)

			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("upstream failed", fault.Code(tt.urn))
			})

			err := handler(context.Background(), sess)
			require.NoError(t, err)

			var body ProblemResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

			require.NotEmpty(t, body.Instance, "JSON error must include instance")
			require.Equal(t, instanceURN(sess.RequestID()), body.Instance,
				"instance must be the request URN (RFC 9457 §3.1.5)")
			require.NotEmpty(t, body.Code)
		})
	}
}

func TestWithObservability_HTMLFallbackIncludesInstance(t *testing.T) {
	t.Parallel()

	// When the HTML renderer fails the middleware falls back to JSON.
	// That fallback must also carry instance.
	mw := WithObservability(renderFunc(func(errorpage.Data) ([]byte, error) {
		return nil, fault.New("template broken")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")

	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Proxy.UpstreamConnectionReset.URN()))
	})

	err := handler(context.Background(), sess)
	require.NoError(t, err)

	var body ProblemResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	require.NotEmpty(t, body.Instance, "JSON fallback must include instance")
	require.Equal(t, instanceURN(sess.RequestID()), body.Instance)
}

type renderFunc func(errorpage.Data) ([]byte, error)

func (f renderFunc) Render(d errorpage.Data) ([]byte, error) { return f(d) }

// TestProblemResponse_GoldenJSON locks the exact JSON envelope that
// HTTP customers receive. Any field rename, addition, removal, or
// type change shows up as a reviewed diff. Field order is locked by
// the ProblemResponse struct definition.
//
// This is the contract third-party SDKs and customer log parsers
// depend on. Treat any change here as a public API change.
func TestProblemResponse_GoldenJSON(t *testing.T) {
	t.Parallel()

	mw := WithObservability(&stubRenderer{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/problem+json")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Auth.RateLimited.URN()))
	})
	require.NoError(t, handler(context.Background(), sess))

	require.Equal(t, 429, w.Code)
	require.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
	require.Equal(t, "30", w.Header().Get("Retry-After"))

	// Parse-and-compare against an expected map. Request ID is
	// per-request so we copy it in rather than baking a literal.
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	// retryable / retry_after are intentionally absent from the body —
	// retry backoff is carried by the standard Retry-After header only
	// (asserted above). See the ProblemResponse doc.
	want := map[string]any{
		"type":     "https://unkey.com/docs/errors/rate-limited",
		"title":    "Too Many Requests",
		"status":   float64(429), // json.Unmarshal → float64
		"detail":   "Rate limit exceeded. Retry later.",
		"instance": instanceURN(sess.RequestID()),
		"code":     "rate_limited",
	}
	require.Equal(t, want, got,
		"problem+json envelope drifted — this is a public API contract")
}
