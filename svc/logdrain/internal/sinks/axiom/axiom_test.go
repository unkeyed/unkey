package axiom_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks/axiom"
)

func TestSink_Send_WireFormat(t *testing.T) {
	t.Parallel()

	var capturedBody []byte
	var capturedHeader http.Header
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ingested": 1}`)
	}))
	defer srv.Close()

	sink := axiom.New(axiom.Config{Dataset: "unkey-test", Endpoint: srv.URL}, "secret-token", srv.Client())

	rec := sinks.Record{
		Kind:          sinks.RecordRuntime,
		TimeMs:        time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC).UnixMilli(),
		SeverityText:  "error",
		Body:          "panic: nil pointer dereference",
		WorkspaceID:   "ws_abc",
		ProjectID:     "proj_def",
		EnvironmentID: "env_ghi",
		AppID:         "app_jkl",
		DeploymentID:  "dep_mno",
		Region:        "us-east-1",
		Platform:      "aws",
		K8sPodName:    "deployment-7d9c-xyz",
		Attributes:    map[string]any{"trace_id": "abc123", "user_id": float64(42)},
	}
	require.NoError(t, sink.Send(context.Background(), []sinks.Record{rec}))

	// URL: dataset path.
	require.Equal(t, "/v1/datasets/unkey-test/ingest", capturedPath)
	// Auth header is the verbatim Bearer.
	require.Equal(t, "Bearer secret-token", capturedHeader.Get("Authorization"))
	require.Equal(t, "application/json", capturedHeader.Get("Content-Type"))

	var events []map[string]any
	require.NoError(t, json.Unmarshal(capturedBody, &events))
	require.Len(t, events, 1)
	ev := events[0]

	require.Equal(t, "2026-04-30T12:00:00Z", ev["_time"])
	require.Equal(t, "error", ev["level"])
	require.Equal(t, "panic: nil pointer dereference", ev["message"])
	require.Equal(t, "ws_abc", ev["workspace_id"])
	require.Equal(t, "dep_mno", ev["deployment_id"])
	require.Equal(t, "runtime", ev["source"])

	attrs, ok := ev["attributes"].(map[string]any)
	require.True(t, ok, "attributes should be a nested object")
	require.Equal(t, "abc123", attrs["trace_id"])
}

func TestSink_Send_ProviderErrorIsSurfaced(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "invalid api token")
	}))
	defer srv.Close()

	sink := axiom.New(axiom.Config{Dataset: "x", Endpoint: srv.URL}, "bad", srv.Client())
	err := sink.Send(context.Background(), []sinks.Record{{TimeMs: 1, Kind: sinks.RecordRuntime}})
	require.Error(t, err)
	// The dashboard shows this verbatim, so the provider's text must reach
	// the caller untouched.
	require.Contains(t, err.Error(), "401")
	require.Contains(t, err.Error(), "invalid api token")
}

func TestSink_Send_EmptyBatchIsNoop(t *testing.T) {
	t.Parallel()

	// Using nil client to prove we never touch HTTP on an empty batch.
	sink := axiom.New(axiom.Config{Dataset: "x"}, "tok", nil)
	require.NoError(t, sink.Send(context.Background(), nil))
	require.NoError(t, sink.Send(context.Background(), []sinks.Record{}))
}
