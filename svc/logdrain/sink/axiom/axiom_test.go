package axiom

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// TestDeliverSuccess guarantees acknowledged events use Axiom NDJSON and that
// dataset names are percent-escaped in the ingest path.
func TestDeliverSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/datasets/a%20dataset/ingest", r.URL.EscapedPath())
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		require.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))
		scanner := bufio.NewScanner(r.Body)
		require.True(t, scanner.Scan())
		var line map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &line))
		require.Equal(t, "1970-01-01T00:00:00.123Z", line["_time"])
		require.Equal(t, "audit_logs", line["stream"])
		event, ok := line["event"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "created", event["action"])
		require.Equal(t, "evt_1", event["id"])
		require.True(t, scanner.Scan())
		require.False(t, scanner.Scan())
		_, err := w.Write([]byte(`{"ingested":2,"failed":0}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	drain, err := New(Config{BaseURL: server.URL, Dataset: "a dataset", Token: "token", UnsafeAllowTestEndpoint: true})
	require.NoError(t, err)
	batch := testBatch()
	expectedBody, err := marshalEvents(batch.Events)
	require.NoError(t, err)
	result, err := drain.Deliver(context.Background(), batch)
	require.NoError(t, err)
	require.True(t, result.Acknowledged)
	require.Equal(t, http.StatusOK, result.HTTPStatus)
	require.Equal(t, int64(len(expectedBody)), result.RequestBodyBytes)
}

// TestRejectedResponses guarantees that HTTP rejections and partial ingestion
// return structured, unacknowledged results.
func TestRejectedResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "reported failure", status: http.StatusOK, body: `{"failed":1}`},
		{name: "bad request", status: http.StatusBadRequest},
		{name: "server error", status: http.StatusInternalServerError},
		{name: "rate limited", status: http.StatusTooManyRequests},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, err := w.Write([]byte(tt.body))
				require.NoError(t, err)
			}))
			t.Cleanup(server.Close)
			drain, err := New(Config{BaseURL: server.URL, Dataset: "dataset", Token: "token", UnsafeAllowTestEndpoint: true})
			require.NoError(t, err)
			result, deliverErr := drain.Deliver(context.Background(), testBatch())
			require.NoError(t, deliverErr)
			require.False(t, result.Acknowledged)
			require.Equal(t, tt.status, result.HTTPStatus)
			require.Equal(t, tt.body, result.ResponseBody)
			require.Positive(t, result.RequestBodyBytes)
		})
	}
}

// TestRetryAfter guarantees Axiom's standard header takes precedence over its
// provider-specific reset timestamp.
func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	t.Run("uses Axiom reset timestamp", func(t *testing.T) {
		headers := http.Header{"X-Ratelimit-Reset": []string{"1787832300"}}
		require.Equal(t, 5*time.Minute, retryAfter(headers, now))
	})
	t.Run("prefers standard Retry-After", func(t *testing.T) {
		headers := http.Header{
			"Retry-After":       []string{"120"},
			"X-Ratelimit-Reset": []string{"1787832300"},
		}
		require.Equal(t, 2*time.Minute, retryAfter(headers, now))
	})
	t.Run("uses reset timestamp after invalid Retry-After", func(t *testing.T) {
		headers := http.Header{
			"Retry-After":       []string{"later"},
			"X-Ratelimit-Reset": []string{"1787832300"},
		}
		require.Equal(t, 5*time.Minute, retryAfter(headers, now))
	})
	t.Run("ignores expired reset timestamp", func(t *testing.T) {
		headers := http.Header{"X-Ratelimit-Reset": []string{"1787831999"}}
		require.Zero(t, retryAfter(headers, now))
	})
}

// TestDeliverTransportFailureReportsBodySize guarantees telemetry can record
// encoded request bytes even when Axiom returns no HTTP response.
func TestDeliverTransportFailureReportsBodySize(t *testing.T) {
	drain, err := New(Config{Dataset: "dataset", Token: "token"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := drain.Deliver(ctx, testBatch())
	require.Error(t, err)
	require.Zero(t, result.HTTPStatus)
	require.Positive(t, result.RequestBodyBytes)
}

// TestNewRejectsInvalidConfig guarantees a missing dataset, missing token, or a
// plain-http BaseURL are rejected at construction.
func TestNewRejectsInvalidConfig(t *testing.T) {
	tests := []Config{{Token: "token"}, {Dataset: "dataset"}, {BaseURL: "http://example.com", Dataset: "dataset", Token: "token"}}
	for _, cfg := range tests {
		_, err := New(cfg)
		require.Error(t, err)
	}
}

// testBatch provides two distinct events so tests can detect dropped or merged NDJSON lines.
func testBatch() sink.Batch {
	return sink.Batch{Events: []sink.Event{
		{EventID: "evt_1", Stream: "audit_logs", Time: 123, Payload: sink.AuditLogPayload{ID: "evt_1", Action: "created"}},
		{EventID: "evt_2", Stream: "audit_logs", Time: 456, Payload: sink.AuditLogPayload{ID: "evt_2", Action: "deleted"}},
	}}
}
