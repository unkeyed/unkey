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

// runClickHouseLoggingRequest runs one request through the ClickHouse logging
// middleware with a handler that populates tracking the way the proxy handler
// does, and returns the rows flushed to the batch processor.
func runClickHouseLoggingRequest(t *testing.T, logRequest bool) []schema.FrontlineRequest {
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
		tracking.LogRequest = logRequest
		return nil
	})

	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil), 0))
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

func TestClickHouseLogging_EmitsRowWhenLoggingPolicyMatched(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, true)

	require.Len(t, rows, 1)
	require.Equal(t, "dep_123", rows[0].DeploymentID)
	require.Equal(t, "/test", rows[0].Path)
}

func TestClickHouseLogging_SkipsRowWithoutLoggingPolicy(t *testing.T) {
	rows := runClickHouseLoggingRequest(t, false)

	require.Empty(t, rows)
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
