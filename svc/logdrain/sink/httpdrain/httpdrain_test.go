package httpdrain

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// TestDeliverSuccess guarantees the default format delivers one JSON array of
// WorkOS-shaped objects with batch metadata in X-Unkey-* headers, and that
// 2xx counts as acknowledgment.
func TestDeliverSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "unkey-logdrain/1", r.Header.Get("User-Agent"))
		require.Equal(t, "customer-value", r.Header.Get("X-Customer"))
		require.Equal(t, "v1", r.Header.Get("X-Unkey-Schema-Version"))
		require.Equal(t, "drain_1", r.Header.Get("X-Unkey-Drain-Id"))
		require.Equal(t, "ws_1", r.Header.Get("X-Unkey-Workspace-Id"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var events []map[string]any
		require.NoError(t, json.Unmarshal(body, &events))
		require.Len(t, events, 2)
		require.Equal(t, "1970-01-01T00:00:00.123Z", events[0]["timestamp"])
		payload, ok := events[0]["event"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "created", payload["action"])
		require.Equal(t, "evt_1", payload["id"])
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	drain := newTestSink(t, Config{Endpoint: server.URL, Headers: http.Header{"X-Customer": {"customer-value"}}, Timeout: time.Second})
	batch := testBatch()
	expectedBody, err := marshalBatch(batch, logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON)
	require.NoError(t, err)
	result, err := drain.Deliver(context.Background(), batch)
	require.NoError(t, err)
	require.True(t, result.Acknowledged)
	require.Equal(t, http.StatusNoContent, result.HTTPStatus)
	require.Equal(t, int64(len(expectedBody)), result.RequestBodyBytes)
}

// TestDeliverNDJSONFormat guarantees the NDJSON format delivers one
// WorkOS-shaped JSON object per line with Content-Type application/x-ndjson.
func TestDeliverNDJSONFormat(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
		require.Len(t, lines, 2)
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &event))
		require.Equal(t, "1970-01-01T00:00:00.123Z", event["timestamp"])
		payload, ok := event["event"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "created", payload["action"])
		require.Equal(t, "evt_1", payload["id"])
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	drain := newTestSink(t, Config{Endpoint: server.URL, Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON, Timeout: time.Second})
	result, err := drain.Deliver(context.Background(), testBatch())
	require.NoError(t, err)
	require.True(t, result.Acknowledged)
}

// TestNewRejectsUnknownFormat guarantees a typo in the stored format returns an
// error instead of silently falling back to NDJSON.
func TestNewRejectsUnknownFormat(t *testing.T) {
	_, err := New(Config{Endpoint: "https://example.com/logs", Format: logdrainv1.HttpBodyFormat(99)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown http drain format")
}

// TestRejectedResponses guarantees that every non-2xx response returns a
// structured, unacknowledged result instead of an operational error.
func TestRejectedResponses(t *testing.T) {
	tests := []struct {
		status int
	}{
		{status: http.StatusInternalServerError},
		{status: http.StatusBadRequest},
		{status: http.StatusTooManyRequests},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "diagnostic", tt.status)
			}))
			t.Cleanup(server.Close)
			result, err := newTestSink(t, Config{Endpoint: server.URL}).Deliver(context.Background(), testBatch())
			require.NoError(t, err)
			require.False(t, result.Acknowledged)
			require.Equal(t, tt.status, result.HTTPStatus)
			require.Equal(t, "diagnostic", result.ResponseBody)
			require.Positive(t, result.RequestBodyBytes)
		})
	}
}

// TestRetryAfterHint guarantees the generic HTTP sink returns a standard
// Retry-After delay for the engine to apply.
func TestRetryAfterHint(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	result, err := newTestSink(t, Config{Endpoint: server.URL}).Deliver(context.Background(), testBatch())
	require.NoError(t, err)
	require.Equal(t, 2*time.Minute, result.RetryAfter)
}

// TestDeliverTransportFailureReportsBodySize guarantees telemetry can record
// encoded request bytes even when the destination returns no HTTP response.
func TestDeliverTransportFailureReportsBodySize(t *testing.T) {
	drain := newTestSink(t, Config{Endpoint: "https://example.com/logs"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := drain.Deliver(ctx, testBatch())
	require.Error(t, err)
	require.Zero(t, result.HTTPStatus)
	require.Positive(t, result.RequestBodyBytes)
}

// TestNewRejectsPlainHTTPEndpoint guarantees plain-http endpoints are rejected
// at construction.
func TestNewRejectsPlainHTTPEndpoint(t *testing.T) {
	_, err := New(Config{Endpoint: "http://example.com/logs"})
	require.Error(t, err)
}

// newTestSink permits the isolated test server while retaining production request behavior.
func newTestSink(t *testing.T, cfg Config) *Sink {
	t.Helper()
	cfg.UnsafeAllowTestEndpoint = true
	drain, err := New(cfg)
	require.NoError(t, err)
	transport, ok := drain.client.Transport.(*http.Transport)
	require.True(t, ok)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // Test servers use ephemeral self-signed certificates.
	return drain
}

// testBatch provides two distinct events so tests can detect dropped or merged NDJSON lines.
func testBatch() sink.Batch {
	return sink.Batch{
		SchemaVersion: "v1", DrainID: "drain_1", WorkspaceID: "ws_1",
		Events: []sink.Event{
			{EventID: "evt_1", Stream: "audit_logs", Time: 123, Payload: sink.AuditLogPayload{ID: "evt_1", Action: "created"}},
			{EventID: "evt_2", Stream: "audit_logs", Time: 456, Payload: sink.AuditLogPayload{ID: "evt_2", Action: "deleted"}},
		},
	}
}
