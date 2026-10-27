package heimdall

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestAPIRequestErrors(t *testing.T) {
	errorsTotal := func() float64 {
		t.Helper()
		families, err := prometheus.DefaultGatherer.Gather()
		require.NoError(t, err)
		for _, family := range families {
			if family.GetName() == "unkey_heimdall_kubernetes_api_request_errors_total" {
				return family.GetMetric()[0].GetCounter().GetValue()
			}
		}
		t.Fatal("API request error metric missing")
		return 0
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/watch" {
			w.WriteHeader(http.StatusOK)
			err := http.NewResponseController(w).Flush()
			if err != nil {
				t.Error(err)
				return
			}
			<-r.Context().Done()
			return
		}
		status, err := strconv.Atoi(r.URL.Path[1:])
		if err != nil {
			t.Error(err)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	transport := apiTransport{RoundTripper: server.Client().Transport}

	for _, tc := range []struct {
		status int
		errors float64
	}{
		{status: http.StatusServiceUnavailable, errors: 1},
		{status: http.StatusTooManyRequests, errors: 1},
		{status: http.StatusForbidden, errors: 1},
		{status: http.StatusOK, errors: 0},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			before := errorsTotal()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/"+strconv.Itoa(tc.status), nil)
			require.NoError(t, err)
			res, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.Equal(t, tc.status, res.StatusCode)
			_, err = io.Copy(io.Discard, res.Body)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			require.Equal(t, before+tc.errors, errorsTotal())
		})
	}

	before := errorsTotal()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/watch", nil)
	require.NoError(t, err)
	res, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, before, errorsTotal(), "a quiet open watch is not an API request failure")
	require.NoError(t, res.Body.Close())
	require.Equal(t, before, errorsTotal())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = transport.RoundTrip(req.WithContext(ctx))
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, before, errorsTotal(), "shutdown is not an API outage")

	ctx, cancel = context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	t.Cleanup(cancel)
	_, err = transport.RoundTrip(req.WithContext(ctx))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, before+1, errorsTotal())

	server.Close()
	_, err = transport.RoundTrip(req)
	require.Error(t, err)
	require.Equal(t, before+2, errorsTotal(), "connection refusal must count even when client-go retries internally")
}
