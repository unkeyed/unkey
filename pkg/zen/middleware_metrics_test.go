package zen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/redaction"
)

// mockEventBuffer captures API requests for testing
type mockEventBuffer struct {
	mu       sync.Mutex
	requests []schema.ApiRequest
}

func (m *mockEventBuffer) Buffer(req schema.ApiRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, req)
}

func (m *mockEventBuffer) getRequests() []schema.ApiRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]schema.ApiRequest{}, m.requests...)
}

func TestWithMetrics_IPAddressExtraction(t *testing.T) {

	tests := []struct {
		name          string
		xForwardedFor string
		remoteAddr    string
		expectedIP    string
	}{
		{
			name:          "X-Forwarded-For with single IP (no port)",
			xForwardedFor: "192.168.1.1",
			remoteAddr:    "10.0.0.1:12345",
			expectedIP:    "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For with IP:port format strips port",
			xForwardedFor: "192.168.1.1:8080",
			remoteAddr:    "10.0.0.1:12345",
			expectedIP:    "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For with multiple IPs (comma-separated)",
			xForwardedFor: "192.168.1.1, 10.0.0.2, 172.16.0.3",
			remoteAddr:    "10.0.0.1:12345",
			expectedIP:    "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For with whitespace around IPs",
			xForwardedFor: "  192.168.1.1  ,  10.0.0.2  ",
			remoteAddr:    "10.0.0.1:12345",
			expectedIP:    "192.168.1.1",
		},
		{
			name:          "Fallback to RemoteAddr when X-Forwarded-For is empty",
			xForwardedFor: "",
			remoteAddr:    "192.168.1.100:54321",
			expectedIP:    "192.168.1.100",
		},
		{
			name:          "RemoteAddr with port gets port stripped",
			xForwardedFor: "",
			remoteAddr:    "10.20.30.40:12345",
			expectedIP:    "10.20.30.40",
		},
		{
			name:          "IPv6 address in X-Forwarded-For",
			xForwardedFor: "2001:db8::1",
			remoteAddr:    "10.0.0.1:12345",
			expectedIP:    "2001:db8::1",
		},
		{
			name:          "IPv6 with brackets and port in RemoteAddr",
			xForwardedFor: "",
			remoteAddr:    "[2001:db8::1]:8080",
			expectedIP:    "2001:db8::1",
		},
		{
			name:          "RemoteAddr without port",
			xForwardedFor: "",
			remoteAddr:    "192.168.1.1",
			expectedIP:    "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventBuffer := &mockEventBuffer{}

			server, err := New(Config{})
			require.NoError(t, err)

			server.RegisterRoute(
				[]Middleware{
					WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
				},
				NewRoute(http.MethodGet, "/ip-test", func(ctx context.Context, s *Session) error {
					return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
				}),
			)

			req := httptest.NewRequest(http.MethodGet, "/ip-test", nil)
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			req.RemoteAddr = tt.remoteAddr

			recorder := httptest.NewRecorder()
			server.Mux().ServeHTTP(recorder, req)

			require.Equal(t, http.StatusOK, recorder.Code)

			requests := eventBuffer.getRequests()
			require.Len(t, requests, 1)
			require.Equal(t, tt.expectedIP, requests[0].IpAddress)
		})
	}
}

func TestWithMetrics_HeaderFiltering(t *testing.T) {

	t.Run("filters out infrastructure headers", func(t *testing.T) {
		eventBuffer := &mockEventBuffer{}

		server, err := New(Config{})
		require.NoError(t, err)

		server.RegisterRoute(
			[]Middleware{
				WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
			},
			NewRoute(http.MethodGet, "/header-test", func(ctx context.Context, s *Session) error {
				return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/header-test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Port", "443")
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		req.Header.Set("X-Amzn-Trace-Id", "Root=1-123-abc")
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		server.Mux().ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)

		requests := eventBuffer.getRequests()
		require.Len(t, requests, 1)

		// Check that infrastructure headers are filtered out
		for _, header := range requests[0].RequestHeaders {
			require.NotContains(t, header, "X-Forwarded-Proto")
			require.NotContains(t, header, "X-Forwarded-Port")
			require.NotContains(t, header, "X-Forwarded-For")
			require.NotContains(t, header, "X-Amzn-Trace-Id")
		}

		// Check that useful headers are still present
		foundUserAgent := false
		foundContentType := false
		for _, header := range requests[0].RequestHeaders {
			if strings.Contains(header, "User-Agent") {
				foundUserAgent = true
			}
			if strings.Contains(header, "Content-Type") {
				foundContentType = true
			}
		}
		require.True(t, foundUserAgent, "User-Agent header should be present")
		require.True(t, foundContentType, "Content-Type header should be present")

		// IP should still be extracted correctly
		require.Equal(t, "192.168.1.1", requests[0].IpAddress)
	})

	t.Run("redacts authorization header", func(t *testing.T) {
		eventBuffer := &mockEventBuffer{}

		server, err := New(Config{})
		require.NoError(t, err)

		server.RegisterRoute(
			[]Middleware{
				WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
			},
			NewRoute(http.MethodGet, "/auth-test", func(ctx context.Context, s *Session) error {
				return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/auth-test", nil)
		req.Header.Set("Authorization", "Bearer secret-token-12345")

		recorder := httptest.NewRecorder()
		server.Mux().ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)

		requests := eventBuffer.getRequests()
		require.Len(t, requests, 1)

		// Check that Authorization is redacted
		foundAuth := false
		for _, header := range requests[0].RequestHeaders {
			if strings.Contains(header, "Authorization") {
				foundAuth = true
				require.Contains(t, header, "[REDACTED]")
				require.NotContains(t, header, "secret-token")
			}
		}
		require.True(t, foundAuth, "Authorization header should be present but redacted")
	})
}

func TestWithMetrics_InternalErrorLogging(t *testing.T) {

	t.Run("logs internal error message when handler returns error", func(t *testing.T) {
		eventBuffer := &mockEventBuffer{}

		server, err := New(Config{})
		require.NoError(t, err)

		// Register a route with metrics and error handling middleware
		// Order matters: metrics wraps error handling, so metrics runs first/last
		server.RegisterRoute(
			[]Middleware{
				WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
				withErrorHandling(),
			},
			NewRoute(http.MethodGet, "/error-test", func(ctx context.Context, s *Session) error {
				return fault.New("something went wrong internally",
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Public("An unexpected error occurred"),
				)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/error-test", nil)
		recorder := httptest.NewRecorder()

		server.Mux().ServeHTTP(recorder, req)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)

		requests := eventBuffer.getRequests()
		require.Len(t, requests, 1)

		// The internal error message should be logged, not the public one
		require.Contains(t, requests[0].Error, "something went wrong internally")
		require.NotContains(t, requests[0].Error, "An unexpected error occurred")
	})

	t.Run("logs chained internal error messages", func(t *testing.T) {
		eventBuffer := &mockEventBuffer{}

		server, err := New(Config{})
		require.NoError(t, err)

		server.RegisterRoute(
			[]Middleware{
				WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
				withErrorHandling(),
			},
			NewRoute(http.MethodGet, "/chained-error", func(ctx context.Context, s *Session) error {
				baseErr := fault.New("database connection failed")
				return fault.Wrap(baseErr,
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Internal("failed to fetch user"),
					fault.Public("Unable to process request"),
				)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/chained-error", nil)
		recorder := httptest.NewRecorder()

		server.Mux().ServeHTTP(recorder, req)

		require.Equal(t, http.StatusInternalServerError, recorder.Code)

		requests := eventBuffer.getRequests()
		require.Len(t, requests, 1)

		// Both internal messages should be in the error chain
		require.Contains(t, requests[0].Error, "failed to fetch user")
		require.Contains(t, requests[0].Error, "database connection failed")
		// Public message should NOT be in the error
		require.NotContains(t, requests[0].Error, "Unable to process request")
	})

	t.Run("no error logged when handler succeeds", func(t *testing.T) {
		eventBuffer := &mockEventBuffer{}

		server, err := New(Config{})
		require.NoError(t, err)

		server.RegisterRoute(
			[]Middleware{
				WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
				withErrorHandling(),
			},
			NewRoute(http.MethodGet, "/success", func(ctx context.Context, s *Session) error {
				return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/success", nil)
		recorder := httptest.NewRecorder()

		server.Mux().ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)

		requests := eventBuffer.getRequests()
		require.Len(t, requests, 1)
		require.Empty(t, requests[0].Error)
	})

	t.Run("logs error for not found responses", func(t *testing.T) {
		eventBuffer := &mockEventBuffer{}

		server, err := New(Config{})
		require.NoError(t, err)

		server.RegisterRoute(
			[]Middleware{
				WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "plaintext", "variables[].value"})),
				withErrorHandling(),
			},
			NewRoute(http.MethodGet, "/not-found-test", func(ctx context.Context, s *Session) error {
				return fault.New("key not found in database",
					fault.Code(codes.UnkeyDataErrorsKeyNotFound),
					fault.Public("The requested key does not exist"),
				)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/not-found-test", nil)
		recorder := httptest.NewRecorder()

		server.Mux().ServeHTTP(recorder, req)

		require.Equal(t, http.StatusNotFound, recorder.Code)

		requests := eventBuffer.getRequests()
		require.Len(t, requests, 1)

		require.Contains(t, requests[0].Error, "key not found in database")
		require.NotContains(t, requests[0].Error, "The requested key does not exist")
	})
}

// The redactor is only useful if the middleware applies it to what it buffers,
// in both directions. Everything else about redaction is covered in
// pkg/redaction; this pins the wiring.
func TestWithMetrics_RedactsBufferedBodies(t *testing.T) {
	eventBuffer := &mockEventBuffer{}

	server, err := New(Config{})
	require.NoError(t, err)

	server.RegisterRoute(
		[]Middleware{
			WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key", "variables[].value"})),
		},
		NewRoute(http.MethodPost, "/redact-test", func(ctx context.Context, s *Session) error {
			var body map[string]any
			require.NoError(t, s.BindBody(&body))

			return s.JSON(http.StatusOK, map[string]any{
				"variables": []map[string]string{
					{"key": "DATABASE_URL", "value": "postgresql://user:RESPONSE_SECRET@host/db"},
				},
			})
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/redact-test", strings.NewReader(`{"apiId":"api_123","key":"REQUEST_SECRET"}`))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.Mux().ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	// The client still receives the real values; only the logged copy changes.
	require.Contains(t, recorder.Body.String(), "RESPONSE_SECRET")

	requests := eventBuffer.getRequests()
	require.Len(t, requests, 1)

	require.NotContains(t, requests[0].RequestBody, "REQUEST_SECRET")
	require.Contains(t, requests[0].RequestBody, `"apiId":"api_123"`)
	require.NotContains(t, requests[0].ResponseBody, "RESPONSE_SECRET")
	require.Contains(t, requests[0].ResponseBody, "[REDACTED]")
}

// Portal routes authenticate with a session cookie, so the token the spec redacts
// from the exchangeCode response would otherwise be written in the clear by
// every request that used it afterwards.
func TestWithMetrics_RedactsCredentialHeaders(t *testing.T) {
	eventBuffer := &mockEventBuffer{}

	server, err := New(Config{})
	require.NoError(t, err)

	server.RegisterRoute(
		[]Middleware{
			WithMetrics(eventBuffer, InstanceInfo{Region: "test-region"}, redaction.New([]string{"key"})),
		},
		NewRoute(http.MethodGet, "/cookie-test", func(ctx context.Context, s *Session) error {
			s.AddHeader("Set-Cookie", "portal_session=ps_RESPONSE_SECRET; HttpOnly")

			return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/cookie-test", nil)
	req.Header.Set("Cookie", "portal_session=ps_REQUEST_SECRET")
	req.Header.Set("Authorization", "Bearer unkey_REQUEST_SECRET")
	req.Header.Set("User-Agent", "unkey-go/1.2.3")

	recorder := httptest.NewRecorder()
	server.Mux().ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	requests := eventBuffer.getRequests()
	require.Len(t, requests, 1)

	for _, header := range requests[0].RequestHeaders {
		require.NotContains(t, header, "REQUEST_SECRET", "request header leaked a credential")
	}
	for _, header := range requests[0].ResponseHeaders {
		require.NotContains(t, header, "RESPONSE_SECRET", "response header leaked a credential")
	}

	require.Contains(t, requests[0].RequestHeaders, "Cookie: [REDACTED]")
	require.Contains(t, requests[0].RequestHeaders, "Authorization: [REDACTED]")
	require.Contains(t, requests[0].RequestHeaders, "User-Agent: unkey-go/1.2.3")
	require.Contains(t, requests[0].ResponseHeaders, "Set-Cookie: [REDACTED]")
}
