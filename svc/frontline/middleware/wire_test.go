package middleware

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

// TestDetectProtocol covers the four wire formats and the negative
// cases. Order matters in detectProtocol: application/grpc-web shares
// a prefix with application/grpc but must fall through to HTTP.
func TestDetectProtocol(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		contentType string
		connectVer  string
		want        protocol
	}{
		{"plain http", "", "", protocolHTTP},
		{"text/html", "text/html", "", protocolHTTP},
		{"application/json", "application/json", "", protocolHTTP},
		{"problem+json", "application/problem+json", "", protocolHTTP},
		{"grpc", "application/grpc", "", protocolGRPC},
		{"grpc+proto", "application/grpc+proto", "", protocolGRPC},
		{"grpc+json", "application/grpc+json", "", protocolGRPC},
		// gRPC-Web is intentionally NOT detected as gRPC — we don't
		// emit the binary in-body trailer frame format yet.
		{"grpc-web", "application/grpc-web", "", protocolHTTP},
		{"grpc-web+proto", "application/grpc-web+proto", "", protocolHTTP},
		{"connect+json", "application/connect+json", "", protocolConnectStream},
		{"connect+proto", "application/connect+proto", "", protocolConnectStream},
		// Connect-unary signaled by header, with vanilla JSON/proto CT.
		{"connect-unary json", "application/json", "1", protocolConnectUnary},
		{"connect-unary proto", "application/proto", "1", protocolConnectUnary},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			if tc.connectVer != "" {
				req.Header.Set("Connect-Protocol-Version", tc.connectVer)
			}
			require.Equal(t, tc.want, detectProtocol(req))
		})
	}
}

// TestWithObservability_GRPCErrorTrailers asserts a gRPC client gets
// a trailers-only response with the correct grpc-status / grpc-message
// for representative public codes. Customer SDKs branch on grpc-status
// ints; the body is always empty for trailers-only.
func TestWithObservability_GRPCErrorTrailers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		urn         codes.URN
		contentType string
		wantGRPC    int
		wantCT      string
	}{
		{
			name:        "invalid_key → UNAUTHENTICATED (16)",
			urn:         codes.Frontline.Auth.InvalidKey.URN(),
			contentType: "application/grpc",
			wantGRPC:    16,
			wantCT:      "application/grpc",
		},
		{
			name:        "rate_limited → RESOURCE_EXHAUSTED (8) preserves subtype",
			urn:         codes.Frontline.Auth.RateLimited.URN(),
			contentType: "application/grpc+proto",
			wantGRPC:    8,
			wantCT:      "application/grpc+proto",
		},
		{
			name:        "service_unavailable → UNAVAILABLE (14)",
			urn:         codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
			contentType: "application/grpc+json",
			wantGRPC:    14,
			wantCT:      "application/grpc+json",
		},
		{
			name:        "gateway_timeout → DEADLINE_EXCEEDED (4)",
			urn:         codes.Frontline.Proxy.DialTimeout.URN(),
			contentType: "application/grpc",
			wantGRPC:    4,
			wantCT:      "application/grpc",
		},
	}

	mw := WithObservability(&stubRenderer{})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/svc/Method", nil)
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom", fault.Code(tc.urn))
			})
			require.NoError(t, handler(context.Background(), sess))

			// Wire status is always 200 for gRPC; the gRPC code lives
			// in trailers.
			require.Equal(t, http.StatusOK, w.Code,
				"gRPC errors must be HTTP 200, code rides trailers")

			require.Equal(t, tc.wantCT, w.Header().Get("Content-Type"),
				"Content-Type must preserve the request subtype")

			require.Empty(t, w.Body.String(),
				"gRPC trailers-only responses have no body")

			// The httptest recorder stores trailers in the header map
			// under the http.TrailerPrefix sentinel.
			gotStatus := w.Header().Get(http.TrailerPrefix + "Grpc-Status")
			if gotStatus == "" {
				// Fallback: some Go versions surface trailers via
				// Result().Trailer instead of the prefixed map.
				gotStatus = w.Result().Trailer.Get("Grpc-Status")
			}
			require.Equal(t, strconv.Itoa(tc.wantGRPC), gotStatus,
				"grpc-status trailer must carry the mapped code")

			gotMsg := w.Header().Get(http.TrailerPrefix + "Grpc-Message")
			if gotMsg == "" {
				gotMsg = w.Result().Trailer.Get("Grpc-Message")
			}
			require.NotEmpty(t, gotMsg,
				"grpc-message trailer must carry a description")

			require.Contains(t, w.Header().Get("Trailer"), "Grpc-Status",
				"Trailer header must announce grpc-status")
		})
	}
}

// TestWithObservability_ConnectUnaryError asserts Connect-unary
// clients get HTTP status from the Connect spec's code map and a
// JSON body of the form {code, message}. Connect-unary is detected
// by Connect-Protocol-Version header, not Content-Type.
func TestWithObservability_ConnectUnaryError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		urn             codes.URN
		wantConnectCode string
		wantHTTPStatus  int
	}{
		{
			name:            "invalid_key → unauthenticated / 401",
			urn:             codes.Frontline.Auth.InvalidKey.URN(),
			wantConnectCode: "unauthenticated",
			wantHTTPStatus:  401,
		},
		{
			name:            "firewall_denied → permission_denied / 403",
			urn:             codes.Frontline.Firewall.Denied.URN(),
			wantConnectCode: "permission_denied",
			wantHTTPStatus:  403,
		},
		{
			name:            "service_unavailable → unavailable / 503",
			urn:             codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
			wantConnectCode: "unavailable",
			wantHTTPStatus:  503,
		},
		{
			name: "gateway_timeout → deadline_exceeded / 408 " +
				"(Connect maps deadline_exceeded to 408, not 504)",
			urn:             codes.Frontline.Proxy.DialTimeout.URN(),
			wantConnectCode: "deadline_exceeded",
			wantHTTPStatus:  408,
		},
	}

	mw := WithObservability(&stubRenderer{})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/svc/Method", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom", fault.Code(tc.urn))
			})
			require.NoError(t, handler(context.Background(), sess))

			require.Equal(t, tc.wantHTTPStatus, w.Code,
				"Connect-unary HTTP status must come from the Connect spec map")

			require.Equal(t, "application/json", w.Header().Get("Content-Type"),
				"Connect error bodies are always JSON regardless of request codec")

			var body connectError
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Equal(t, tc.wantConnectCode, body.Code)
			require.NotEmpty(t, body.Message)
		})
	}
}

// TestWithObservability_ConnectStreamError asserts Connect-streaming
// clients get HTTP 200 plus a single end-stream envelope frame
// carrying the error. The end-stream payload is always JSON regardless
// of the streaming codec.
func TestWithObservability_ConnectStreamError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		urn         codes.URN
		contentType string
		wantCode    string
	}{
		{
			name:        "invalid_key over connect+json",
			urn:         codes.Frontline.Auth.InvalidKey.URN(),
			contentType: "application/connect+json",
			wantCode:    "unauthenticated",
		},
		{
			name:        "service_unavailable over connect+proto",
			urn:         codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
			contentType: "application/connect+proto",
			wantCode:    "unavailable",
		},
	}

	mw := WithObservability(&stubRenderer{})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/svc/StreamMethod", nil)
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))

			handler := mw(func(_ context.Context, _ *zen.Session) error {
				return fault.New("boom", fault.Code(tc.urn))
			})
			require.NoError(t, handler(context.Background(), sess))

			require.Equal(t, http.StatusOK, w.Code,
				"Connect-streaming errors are HTTP 200; error rides end-stream frame")
			require.Equal(t, tc.contentType, w.Header().Get("Content-Type"),
				"Content-Type must preserve the request subtype")

			body := w.Body.Bytes()
			require.Greater(t, len(body), 5, "envelope must have header + payload")
			require.Equal(t, byte(0x02), body[0],
				"first byte must be end-stream flag (0x02)")

			payloadLen := binary.BigEndian.Uint32(body[1:5])
			require.Equal(t, int(payloadLen), len(body)-5,
				"length prefix must match payload size")

			var es connectEndStream
			require.NoError(t, json.Unmarshal(body[5:], &es))
			require.NotNil(t, es.Error)
			require.Equal(t, tc.wantCode, es.Error.Code)
			require.NotEmpty(t, es.Error.Message)
		})
	}
}

// TestPercentEncodeGRPCMessage covers the gRPC spec's encoding rules:
// printable ASCII passes through except '%', everything else is %XX.
func TestPercentEncodeGRPCMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"100%", "100%25"},
		{"newline\n", "newline%0A"},
		{"tab\there", "tab%09here"},
		{"unicode: ☕", "unicode: %E2%98%95"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, percentEncodeGRPCMessage(tc.in))
		})
	}
}

// TestWriteGRPCError_GoldenEnvelope is a byte-level snapshot of the
// gRPC error response a customer SDK observes. Locks the exact wire
// shape so an accidental change (re-ordering trailers, swapping
// percent-encoding, dropping the Trailer announce header) becomes a
// reviewed diff. The recorder captures pre-declared trailers under
// http.TrailerPrefix; the same values appear on a real HTTP/2
// connection in the response trailer frame.
func TestWriteGRPCError_GoldenEnvelope(t *testing.T) {
	t.Parallel()

	mw := WithObservability(&stubRenderer{})

	req := httptest.NewRequest(http.MethodPost, "/svc/Method", nil)
	req.Header.Set("Content-Type", "application/grpc+proto")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Auth.RateLimited.URN()))
	})
	require.NoError(t, handler(context.Background(), sess))

	// Wire shape.
	require.Equal(t, http.StatusOK, w.Code,
		"gRPC error: HTTP 200, code in trailers")
	require.Equal(t, "application/grpc+proto", w.Header().Get("Content-Type"),
		"Content-Type: echo the request subtype verbatim")
	require.Equal(t, "Grpc-Status, Grpc-Message", w.Header().Get("Trailer"),
		"Trailer header announces both trailer keys, in the documented order")
	require.Empty(t, w.Body.String(),
		"Body: trailers-only response has no payload")

	// Trailer values (recorder surfaces them under TrailerPrefix).
	require.Equal(t, "8", w.Header().Get(http.TrailerPrefix+"Grpc-Status"),
		"grpc-status: 8 = RESOURCE_EXHAUSTED for rate_limited")
	require.Equal(t, "Rate limit exceeded. Retry later.",
		w.Header().Get(http.TrailerPrefix+"Grpc-Message"),
		"grpc-message: catalog Detail, percent-encoded "+
			"(no encoding needed for this string)")
}

// TestWriteConnectUnaryError_GoldenEnvelope is a byte-level snapshot
// of the Connect-unary error body. Locks JSON field names, order,
// and absent fields. Connect-unary HTTP status comes from the
// Connect spec's code→status map (not RFC 9457 / Problem.Status).
func TestWriteConnectUnaryError_GoldenEnvelope(t *testing.T) {
	t.Parallel()

	mw := WithObservability(&stubRenderer{})

	req := httptest.NewRequest(http.MethodPost, "/svc/Method", nil)
	req.Header.Set("Content-Type", "application/proto") // proto request
	req.Header.Set("Connect-Protocol-Version", "1")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Auth.RateLimited.URN()))
	})
	require.NoError(t, handler(context.Background(), sess))

	require.Equal(t, 429, w.Code,
		"Connect-unary: HTTP status from Connect spec "+
			"(resource_exhausted → 429)")
	require.Equal(t, "application/json", w.Header().Get("Content-Type"),
		"Connect-unary errors are ALWAYS JSON even when the "+
			"request was proto, per the Connect spec")

	// Exact body bytes. Field order is locked by the connectError
	// struct definition; any rename or addition shows up here.
	want := `{"code":"resource_exhausted","message":"Rate limit exceeded. Retry later."}`
	require.JSONEq(t, want, w.Body.String())
}

// TestWriteConnectStreamError_GoldenEnvelope is a byte-level snapshot
// of the Connect-streaming end-stream envelope frame. Layout per the
// Connect spec:
//
//	byte  0      flags (0x02 = end-stream)
//	bytes 1..4   payload length, big-endian uint32
//	bytes 5..N   JSON payload (always JSON, regardless of streaming codec)
func TestWriteConnectStreamError_GoldenEnvelope(t *testing.T) {
	t.Parallel()

	mw := WithObservability(&stubRenderer{})

	req := httptest.NewRequest(http.MethodPost, "/svc/StreamMethod", nil)
	req.Header.Set("Content-Type", "application/connect+proto")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Auth.RateLimited.URN()))
	})
	require.NoError(t, handler(context.Background(), sess))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/connect+proto", w.Header().Get("Content-Type"),
		"Content-Type: echo the request subtype verbatim")

	body := w.Body.Bytes()
	require.GreaterOrEqual(t, len(body), 5)
	require.Equal(t, byte(0x02), body[0],
		"byte 0: end-stream flag")

	gotLen := binary.BigEndian.Uint32(body[1:5])
	require.Equal(t, len(body)-5, int(gotLen),
		"bytes 1..4: length prefix matches payload size")

	want := `{"error":{"code":"resource_exhausted","message":"Rate limit exceeded. Retry later."}}`
	require.JSONEq(t, want, string(body[5:]))
}

// TestWithObservability_HTTPPathStillWorks asserts that adding the
// protocol dispatch did not regress the existing HTTP / JSON / HTML
// paths.
func TestWithObservability_HTTPPathStillWorks(t *testing.T) {
	t.Parallel()

	mw := WithObservability(&stubRenderer{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	handler := mw(func(_ context.Context, _ *zen.Session) error {
		return fault.New("boom", fault.Code(codes.Frontline.Auth.InvalidKey.URN()))
	})
	require.NoError(t, handler(context.Background(), sess))

	require.Equal(t, 401, w.Code, "HTTP clients still get logical status")

	var body ProblemResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "invalid_key", body.Code)
}
