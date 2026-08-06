package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

type captureFlags struct {
	requestHeaders  bool
	responseHeaders bool
	requestBody     bool
	responseBody    bool
}

// runClickHouseLoggingRequest runs one request through the ClickHouse logging
// middleware with a handler that populates tracking the way the proxy handler
// does, and returns the rows flushed to the batch processor.
func runClickHouseLoggingRequest(t *testing.T, capture captureFlags) []schema.FrontlineRequest {
	t.Helper()

	var rows []schema.FrontlineRequest
	done := make(chan struct{})
	buf := batch.New(batch.Config[schema.FrontlineRequest]{
		Name:          "test",
		Drop:          false,
		BatchSize:     10,
		BufferSize:    10,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, b []schema.FrontlineRequest) {
			rows = append(rows, b...)
		},
	})

	mw := WithClickHouseLogging(buf, clock.NewTestClock(), "fl_test", "us-east-1", "test")
	handler := mw(func(ctx context.Context, s *zen.Session) error {
		tracking, ok := proxy.RequestTrackingFromContext(ctx)
		require.True(t, ok)
		tracking.DeploymentID = "dep_123"
		tracking.InstanceID = "inst_123"
		tracking.LogRequestHeaders = capture.requestHeaders
		tracking.LogResponseHeaders = capture.responseHeaders
		tracking.LogRequestBody = capture.requestBody
		tracking.LogResponseBody = capture.responseBody
		if capture.requestBody {
			// The proxy handler's TeeReader only captures the body when
			// LogRequestBody is set; emulate that here.
			tracking.RequestBody = []byte(`{"hello":"world"}`)
		}
		if capture.responseBody {
			// The proxy only captures the upstream body when
			// LogResponseBody is set; emulate that here.
			tracking.ResponseBody = []byte(`{"status":"ok"}`)
		}
		s.ResponseWriter().Header().Set("X-Upstream", "upstream-value")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test?page=2", nil)
	req.Header.Set("X-Custom", "custom-value")
	req.Header.Set("User-Agent", "test-agent/1.0")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	require.NoError(t, handler(context.Background(), sess))

	go func() {
		buf.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for batch processor to flush")
	}
	return rows
}

// TestClickHouseLogging_AlwaysEmitsBaseRow pins that the base row is written
// even without any logging policy: the traffic and latency charts depend on
// it. Headers, query data, and bodies must stay empty without an opt-in.
func TestClickHouseLogging_AlwaysEmitsBaseRow(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, captureFlags{})

	require.Len(t, rows, 1)
	require.Equal(t, "dep_123", rows[0].DeploymentID)
	require.Equal(t, "/test", rows[0].Path)
	require.Empty(t, rows[0].QueryString)
	require.Empty(t, rows[0].QueryParams)
	require.Empty(t, rows[0].RequestHeaders)
	require.Empty(t, rows[0].ResponseHeaders)
	require.Empty(t, rows[0].RequestBody)
	require.Empty(t, rows[0].ResponseBody)
	require.Empty(t, rows[0].UserAgent, "user agent identifies the client and rides on the request-headers opt-in")
	require.Empty(t, rows[0].IPAddress, "client IP identifies the client and rides on the request-headers opt-in")
}

func TestClickHouseLogging_RequestHeadersCaptureIsOptIn(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, captureFlags{requestHeaders: true})

	require.Len(t, rows, 1)
	require.Equal(t, "page=2", rows[0].QueryString)
	require.Contains(t, rows[0].RequestHeaders, "X-Custom: custom-value")
	require.Equal(t, "test-agent/1.0", rows[0].UserAgent)
	require.Empty(t, rows[0].ResponseHeaders, "response headers are a separate opt-in")
	require.Empty(t, rows[0].RequestBody)
}

func TestClickHouseLogging_ResponseHeadersCaptureIsOptIn(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, captureFlags{responseHeaders: true})

	require.Len(t, rows, 1)
	require.Empty(t, rows[0].RequestHeaders, "request headers are a separate opt-in")
	require.Empty(t, rows[0].QueryString, "query data belongs to request headers capture")
	require.Contains(t, rows[0].ResponseHeaders, "X-Upstream: upstream-value")
}

func TestClickHouseLogging_RequestBodyCaptureIsOptIn(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, captureFlags{requestBody: true})

	require.Len(t, rows, 1)
	require.Equal(t, `{"hello":"world"}`, rows[0].RequestBody)
	require.Empty(t, rows[0].ResponseBody, "response body is a separate opt-in")
	require.Empty(t, rows[0].RequestHeaders)
	require.Empty(t, rows[0].QueryString)
}

func TestClickHouseLogging_ResponseBodyCaptureIsOptIn(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, captureFlags{responseBody: true})

	require.Len(t, rows, 1)
	require.Equal(t, `{"status":"ok"}`, rows[0].ResponseBody)
	require.Empty(t, rows[0].RequestBody, "request body is a separate opt-in")
	require.Empty(t, rows[0].RequestHeaders)
	require.Empty(t, rows[0].ResponseHeaders)
}

func TestFormatHeaders_RedactsAuthorization(t *testing.T) {
	h := http.Header{
		"Authorization": []string{"Bearer sk_live_secret"},
		"Content-Type":  []string{"application/json"},
	}

	got := formatHeaders(h, nil)

	require.Contains(t, got, "Authorization: [REDACTED]")
	require.Contains(t, got, "Content-Type: application/json")
	require.NotContains(t, got, "sk_live_secret")
}

func TestFormatHeaders_RedactsConfiguredSecretHeaders(t *testing.T) {
	h := http.Header{
		"X-Api-Key":    []string{"sk_live_secret"},
		"Content-Type": []string{"application/json"},
	}

	// Secret names are lowercased, matching http.Header's canonicalized keys
	// after ToLower.
	got := formatHeaders(h, map[string]struct{}{"x-api-key": {}})

	require.Contains(t, got, "X-Api-Key: [REDACTED]")
	require.Contains(t, got, "Content-Type: application/json")
	require.NotContains(t, got, "sk_live_secret")
}

func TestFormatHeaders_NoSecretsLeavesValues(t *testing.T) {
	h := http.Header{"X-Api-Key": []string{"sk_live_secret"}}

	got := formatHeaders(h, nil)

	require.Contains(t, got, "X-Api-Key: sk_live_secret")
}

func TestRedactQueryParams_RedactsConfigured(t *testing.T) {
	values := url.Values{
		"api_key": []string{"sk_live_secret"},
		"page":    []string{"2"},
	}

	got := redactQueryParams(values, map[string]struct{}{"api_key": {}})

	require.Equal(t, []string{"[REDACTED]"}, got["api_key"])
	require.Equal(t, []string{"2"}, got["page"])
	// Input is not mutated.
	require.Equal(t, []string{"sk_live_secret"}, values["api_key"])
}

func TestToSet(t *testing.T) {
	require.Nil(t, toSet(nil))
	require.Nil(t, toSet([]string{}))

	set := toSet([]string{"a", "b"})
	require.Contains(t, set, "a")
	require.Contains(t, set, "b")
}
