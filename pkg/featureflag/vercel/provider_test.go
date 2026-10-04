package vercel

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
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

func TestRunStopsWhenContextEnds(t *testing.T) {
	body := []byte(`{"environment":"production","definitions":{}}`)
	p := testProvider(t, body, body)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context ended")
	}
}
