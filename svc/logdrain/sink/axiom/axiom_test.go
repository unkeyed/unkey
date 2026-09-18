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

func TestDeliverRatelimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var record map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&record))
		require.JSONEq(t, `"ratelimits"`, string(record["stream"]))
		require.JSONEq(t, `"1970-01-01T00:00:00.123Z"`, string(record["_time"]))
		require.Equal(t, record["_time"], record["time"])
		body, err := json.Marshal(record)
		require.NoError(t, err)
		require.JSONEq(t, `{"stream":"ratelimits","time":"1970-01-01T00:00:00.123Z","_time":"1970-01-01T00:00:00.123Z","request_id":"req","namespace_id":"ns","identifier":"customer\n1","passed":true,"override_id":"override","limit":100,"remaining":97,"tokens":3,"reset_at":10000,"source":"api"}`, string(body))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	batch := testBatch()
	batch.Events = []sink.Event{{EventID: "req", Stream: "ratelimits", Time: 123, Payload: sink.RatelimitPayload{RequestID: "req", NamespaceID: "ns", Identifier: "customer\n1", Passed: true, OverrideID: "override", Limit: 100, Remaining: 97, Tokens: 3, ResetAt: 10000, Source: "api"}}}
	result, err := newTestDrain(t, server.URL, "decisions", "token").Deliver(t.Context(), batch)
	require.NoError(t, err)
	require.True(t, result.Acknowledged)
}

func TestDeliverRuntimeLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var record map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&record))
		require.JSONEq(t, `"runtime_logs"`, string(record["stream"]))
		require.JSONEq(t, `"1970-01-01T00:00:00.123Z"`, string(record["_time"]))
		body, err := json.Marshal(record)
		require.NoError(t, err)
		require.JSONEq(t, `{"stream":"runtime_logs","time":"1970-01-01T00:00:00.123Z","_time":"1970-01-01T00:00:00.123Z","log_id":"rlog_1","severity":"error","message":"first\nsecond","attributes":{"order":{"id":42}},"project_id":"project","app_id":"app","environment_id":"env","deployment_id":"deployment","region":"local"}`, string(body))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	batch := testBatch()
	batch.Events = []sink.Event{{EventID: "rlog_1", Stream: "runtime_logs", Time: 123, Payload: sink.RuntimeLogPayload{LogID: "rlog_1", Severity: "error", Message: "first\nsecond", Attributes: json.RawMessage(`{"order":{"id":42}}`), ProjectID: "project", AppID: "app", EnvironmentID: "env", DeploymentID: "deployment", Region: "local"}}}
	result, err := newTestDrain(t, server.URL, "runtime", "token").Deliver(t.Context(), batch)
	require.NoError(t, err)
	require.True(t, result.Acknowledged)
}

func TestDeliverGatewayRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var line map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&line))
		require.Equal(t, "1970-01-01T00:00:00.123Z", line["_time"])
		require.Equal(t, line["_time"], line["time"])
		require.NotContains(t, line, "event")
		require.NotContains(t, line, "timestamp")
		require.Equal(t, "gateway_requests", line["stream"])
		require.Equal(t, "req_gateway", line["request_id"])
		require.Equal(t, map[string]any{"status": float64(503), "headers": []any{"Content-Type: application/json"}, "body": "response\nbody"}, line["response"])
		require.Equal(t, map[string]any{"total": float64(0), "instance": float64(41), "gateway": float64(0)}, line["latency"])
		request, ok := line["request"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, map[string]any{"tag": []any{"a", "b"}}, request["query_params"])
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	batch := testBatch()
	batch.Events = []sink.Event{{EventID: "req_gateway", Stream: "gateway_requests", Time: 123, Payload: sink.GatewayRequestPayload{RequestID: "req_gateway", Response: sink.GatewayResponse{Status: 503, Headers: []string{"Content-Type: application/json"}, Body: "response\nbody"}, Latency: sink.GatewayRequestLatency{Instance: 41}, Request: sink.GatewayRequest{QueryParams: map[string][]string{"tag": {"a", "b"}}}}}}
	result, err := newTestDrain(t, server.URL, "gateway", "token").Deliver(t.Context(), batch)
	require.NoError(t, err)
	require.True(t, result.Acknowledged)
}

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
		require.Equal(t, line["_time"], line["time"])
		require.NotContains(t, line, "event")
		require.Empty(t, r.URL.RawQuery)
		event := line
		require.Equal(t, "created", event["action"])
		require.Equal(t, "evt_1", event["id"])
		require.True(t, scanner.Scan())
		var verification map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &verification))
		require.Equal(t, "key_verifications", verification["stream"])
		require.Equal(t, "1970-01-01T00:00:00.456Z", verification["time"])
		require.Equal(t, verification["time"], verification["_time"])
		require.Equal(t, "req_2", verification["request_id"])
		require.Equal(t, "VALID", verification["outcome"])
		require.NotContains(t, verification, "event")
		require.NotContains(t, verification, "timestamp")
		require.False(t, scanner.Scan())
		_, err := w.Write([]byte(`{"ingested":2,"failed":0}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	drain := newTestDrain(t, server.URL, "a dataset", "token")
	batch := testBatch()
	batch.Events[1] = sink.Event{EventID: "req_2", Stream: "key_verifications", Time: 456, Payload: sink.KeyVerificationPayload{RequestID: "req_2", Outcome: "VALID"}}
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
			drain := newTestDrain(t, server.URL, "dataset", "token")
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

// TestNewRejectsInvalidConfig guarantees a missing dataset or token is rejected.
func TestNewRejectsInvalidConfig(t *testing.T) {
	tests := []Config{{Token: "token"}, {Dataset: "dataset"}}
	for _, cfg := range tests {
		_, err := New(cfg)
		require.Error(t, err)
	}
}

func newTestDrain(t *testing.T, baseURL, dataset, token string) *Sink {
	t.Helper()
	drain, err := New(Config{Dataset: dataset, Token: token})
	require.NoError(t, err)
	drain.client.Transport = rewriteHost{baseURL: baseURL, next: http.DefaultTransport}
	return drain
}

type rewriteHost struct {
	baseURL string
	next    http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	stub, err := http.NewRequestWithContext(req.Context(), req.Method, r.baseURL+req.URL.RequestURI(), req.Body)
	if err != nil {
		return nil, err
	}
	stub.Header = req.Header.Clone()
	return r.next.RoundTrip(stub)
}

// testBatch provides two distinct events so tests can detect dropped or merged NDJSON lines.
func testBatch() sink.Batch {
	return sink.Batch{Events: []sink.Event{
		{EventID: "evt_1", Stream: "audit_logs", Time: 123, Payload: sink.AuditLogPayload{ID: "evt_1", Action: "created"}},
		{EventID: "evt_2", Stream: "audit_logs", Time: 456, Payload: sink.AuditLogPayload{ID: "evt_2", Action: "deleted"}},
	}}
}
