package vercel_test

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
	"github.com/unkeyed/unkey/pkg/featureflag/vercel"
)

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
	p, err := vercel.NewWithEndpoint(vercel.Config{SDKKey: "vf_server_test", RefreshInterval: time.Second, HTTPTimeout: 0, MaxStaleness: 0}, server.URL, server.Client())
	require.NoError(t, err)
	api, err := featureflag.New(t.Context(), p)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Shutdown(context.Background())) })
	runCtx, stopRun := context.WithCancel(t.Context())
	runDone := make(chan error, 1)
	go func() { runDone <- p.Run(runCtx) }()
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
	stopRun()
	require.NoError(t, <-runDone)
}
