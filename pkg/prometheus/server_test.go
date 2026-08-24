package prometheus

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	clientprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewWithRegistryExposesMetrics(t *testing.T) {
	reg := clientprometheus.NewRegistry()
	gauge := clientprometheus.NewGauge(clientprometheus.GaugeOpts{
		Name: "test_metric",
		Help: "Test metric.",
	})
	gauge.Set(42)
	reg.MustRegister(gauge)

	server, err := NewWithRegistry(reg)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			require.NoError(t, closeErr)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(ctx, listener)
	}()

	response, err := http.Get("http://" + listener.Addr().String() + "/metrics")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, response.Body.Close())
	})
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), "test_metric 42")

	cancel()
	select {
	case err := <-serveErr:
		require.NoError(t, err)
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for metrics server to stop")
	}
}

func TestNewWithRegistryRejectsNilRegistry(t *testing.T) {
	_, err := NewWithRegistry(nil)
	require.EqualError(t, err, "prometheus: nil registry")
}
