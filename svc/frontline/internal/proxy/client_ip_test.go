package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

func TestForwardToInstanceReplacesSpoofedForwardedFor(t *testing.T) {
	t.Parallel()

	const clientIP = "198.51.100.42"
	var forwarded http.Header
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		forwarded = req.Header.Clone()
		return &http.Response{ //nolint:exhaustruct
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       io.NopCloser(http.NoBody),
			Request:    req,
		}, nil
	})
	transports := &TransportRegistry{
		transports: map[db.DeploymentsUpstreamProtocol]http.RoundTripper{
			db.DeploymentsUpstreamProtocolHttp1: transport,
		},
		fallback: transport,
	}
	clk := clock.NewTestClock(time.Now())
	service, err := New(Config{ //nolint:exhaustruct
		InstanceID:         "frontline_test",
		Platform:           "aws",
		Region:             "us-east-1",
		Clock:              clk,
		UpstreamTransports: transports,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "https://customer.example/path", nil)
	req.RemoteAddr = clientIP + ":12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.77")
	recorder := httptest.NewRecorder()
	sess := &zen.Session{} //nolint:exhaustruct
	require.NoError(t, sess.Init(recorder, req, 0))

	ctx := WithRequestStartTime(context.Background(), clk.Now())
	err = service.ForwardToInstance(ctx, sess, db.DeploymentsUpstreamProtocolHttp1, db.FindInstancesByDeploymentIDRow{ //nolint:exhaustruct
		Address: "customer.internal:8080",
	})
	require.NoError(t, err)
	require.Equal(t, clientIP, forwarded.Get("X-Forwarded-For"))
	require.Equal(t, "customer.example", forwarded.Get("X-Forwarded-Host"))
	require.Equal(t, "https", forwarded.Get("X-Forwarded-Proto"))
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
