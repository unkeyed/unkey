package vercel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/featureflag"
)

func TestInitializationAuthAndShutdown(t *testing.T) {
	body := []byte(`{"environment":"production","definitions":{}}`)
	var authorization string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		authorization = req.Header.Get("Authorization")
		return response(body), nil
	})}
	p, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: time.Second, HTTPTimeout: 0, MaxStaleness: time.Second}, "https://example.invalid/data", client)
	require.NoError(t, err)
	require.NoError(t, p.InitWithContext(t.Context(), openfeature.EvaluationContext{}))
	require.Equal(t, "Bearer vf_server_test", authorization)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	require.NoError(t, p.ShutdownWithContext(ctx))
	require.NoError(t, p.ShutdownWithContext(ctx))
}

func TestPollingRecoversFromColdFailureThroughOpenFeature(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	var healthy atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Authorization") != "Bearer vf_server_test" {
			t.Error("missing SDK authentication")
		}
		if _, err := w.Write(body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	p, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: time.Second, HTTPTimeout: 0, MaxStaleness: 0}, server.URL, server.Client())
	require.NoError(t, err)
	api, err := featureflag.New(t.Context(), p)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Shutdown(context.Background())) })
	client := api.NewClient()
	evaluation := openfeature.NewTargetlessEvaluationContext(map[string]any{"team": map[string]any{"id": "matching"}})
	detail, err := client.BooleanValueDetails(t.Context(), "direct", false, evaluation)
	require.Error(t, err)
	require.False(t, detail.Value)
	require.Equal(t, "http_401", p.Diagnostics().LastRefreshError)
	require.Positive(t, p.Diagnostics().RefreshFailures)
	healthy.Store(true)
	require.Eventually(t, func() bool {
		value, err := client.BooleanValueDetails(t.Context(), "direct", false, evaluation)
		return err == nil && value.Value
	}, 5*time.Second, 10*time.Millisecond)
	require.True(t, p.Diagnostics().Ready)
	require.Empty(t, p.Diagnostics().LastRefreshError)
	require.NoError(t, api.Shutdown(t.Context()))
	select {
	case <-p.stopped:
	default:
		t.Fatal("polling continues after shutdown")
	}
}

func TestShutdownWithoutContextStopsPolling(t *testing.T) {
	body := []byte(`{"environment":"production","definitions":{}}`)
	p := testProvider(t, body, body, body)
	require.NoError(t, p.InitWithContext(t.Context(), openfeature.EvaluationContext{}))

	p.Shutdown()
	p.Shutdown()

	select {
	case <-p.stopped:
	default:
		t.Fatal("polling continues after Shutdown")
	}
}
