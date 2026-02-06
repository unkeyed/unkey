package runner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

func checkHealthEndpoint(t *testing.T, mux *http.ServeMux, endpoint string, wantCode int, wantStatus string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, endpoint, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, wantCode, rec.Code, "endpoint %s", endpoint)

	var resp healthResponse
	unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, unmarshalErr)
	require.Equal(t, wantStatus, resp.Status, "endpoint %s", endpoint)
}

func TestHealthEndpoints_Lifecycle(t *testing.T) {
	r := New(logging.NewNoop())
	mux := http.NewServeMux()
	r.RegisterHealth(mux, "/health")

	t.Run("before started all return 503", func(t *testing.T) {
		checkHealthEndpoint(t, mux, "/health/live", http.StatusServiceUnavailable, "not started")
		checkHealthEndpoint(t, mux, "/health/ready", http.StatusServiceUnavailable, "not started")
		checkHealthEndpoint(t, mux, "/health/startup", http.StatusServiceUnavailable, "not started")
	})

	r.health.started.Store(true)

	t.Run("after started all return 200", func(t *testing.T) {
		checkHealthEndpoint(t, mux, "/health/live", http.StatusOK, "ok")
		checkHealthEndpoint(t, mux, "/health/ready", http.StatusOK, "ok")
		checkHealthEndpoint(t, mux, "/health/startup", http.StatusOK, "ok")
	})

	r.health.shuttingDown.Store(true)

	t.Run("during shutdown ready returns 503 but live and startup still 200", func(t *testing.T) {
		checkHealthEndpoint(t, mux, "/health/live", http.StatusOK, "ok")
		checkHealthEndpoint(t, mux, "/health/ready", http.StatusServiceUnavailable, "shutting down")
		checkHealthEndpoint(t, mux, "/health/startup", http.StatusOK, "ok")
	})
}

func TestHealthReady_WithChecks_AllPass(t *testing.T) {
	r := New(logging.NewNoop())
	mux := http.NewServeMux()
	r.RegisterHealth(mux, "/health")
	r.health.started.Store(true)

	r.AddReadinessCheck("database", func(ctx context.Context) error {
		return nil
	})
	r.AddReadinessCheck("redis", func(ctx context.Context) error {
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp healthResponse
	unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, unmarshalErr)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "ok", resp.Checks["database"])
	require.Equal(t, "ok", resp.Checks["redis"])
}

func TestHealthReady_WithChecks_OneFails(t *testing.T) {
	r := New(logging.NewNoop())
	mux := http.NewServeMux()
	r.RegisterHealth(mux, "/health")
	r.health.started.Store(true)

	r.AddReadinessCheck("database", func(ctx context.Context) error {
		return nil
	})
	r.AddReadinessCheck("redis", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp healthResponse
	unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, unmarshalErr)
	require.Equal(t, "fail", resp.Status)
	require.Equal(t, "ok", resp.Checks["database"])
	require.Equal(t, "connection refused", resp.Checks["redis"])
}

func TestHealthReady_CheckTimeout(t *testing.T) {
	r := New(logging.NewNoop())
	mux := http.NewServeMux()
	r.RegisterHealth(mux, "/health")
	r.health.started.Store(true)

	r.AddReadinessCheck("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			return nil
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp healthResponse
	unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, unmarshalErr)
	require.Equal(t, "fail", resp.Status)
	require.Contains(t, resp.Checks["slow"], "context deadline exceeded")
}

func TestAddReadinessCheck_PanicsOnEmptyName(t *testing.T) {
	r := New(logging.NewNoop())
	require.Panics(t, func() {
		r.AddReadinessCheck("", func(ctx context.Context) error { return nil })
	})
}

func TestAddReadinessCheck_PanicsOnNilCheck(t *testing.T) {
	r := New(logging.NewNoop())
	require.Panics(t, func() {
		r.AddReadinessCheck("test", nil)
	})
}

func TestHealthReady_ChecksRunInParallel(t *testing.T) {
	r := New(logging.NewNoop())
	mux := http.NewServeMux()
	r.RegisterHealth(mux, "/health")
	r.health.started.Store(true)

	checkDuration := 50 * time.Millisecond

	r.AddReadinessCheck("check1", func(ctx context.Context) error {
		time.Sleep(checkDuration)
		return nil
	})
	r.AddReadinessCheck("check2", func(ctx context.Context) error {
		time.Sleep(checkDuration)
		return nil
	})
	r.AddReadinessCheck("check3", func(ctx context.Context) error {
		time.Sleep(checkDuration)
		return nil
	})

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Less(t, elapsed, checkDuration*2, "checks should run in parallel")
}
