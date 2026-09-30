package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
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
	metadata, err := meta.New(testMetadataSigningKey)
	require.NoError(t, err)
	service, err := New(Config{ //nolint:exhaustruct
		InstanceID:         "frontline_test",
		Platform:           "aws",
		Region:             "us-east-1",
		Clock:              clk,
		Metadata:           metadata,
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

	sess.SetClientIP(netip.MustParseAddr("2001:db8::42"))
	err = service.ForwardToInstance(ctx, sess, db.DeploymentsUpstreamProtocolHttp1, db.FindInstancesByDeploymentIDRow{ //nolint:exhaustruct
		Address: "customer.internal:8080",
	})
	require.NoError(t, err)
	require.Equal(t, "2001:db8::42", forwarded.Get("X-Forwarded-For"))
}

func TestForwardPreservesRequestAndResponse(t *testing.T) {
	const requestURI = "/items/a%2Fb?q=a+b&q=a%20b&token=%2B%2F%3D&empty="
	body := []byte{0, 1, 255, '\r', '\n', '&', '='}
	type receivedRequest struct {
		uri, host, method, contentType string
		body                           []byte
		err                            error
	}
	for _, destination := range []string{"instance", "region"} {
		t.Run(destination, func(t *testing.T) {
			received := make(chan receivedRequest, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				received <- receivedRequest{r.RequestURI, r.Host, r.Method, r.Header.Get("Content-Type"), data, err}
				w.Header().Set("X-Customer-Response", "preserved")
				w.WriteHeader(http.StatusCreated)
			}))
			t.Cleanup(upstream.Close)
			target, err := url.Parse(upstream.URL)
			require.NoError(t, err)
			svc := &service{clock: clock.NewTestClock(), instanceID: "test", platform: "aws", region: "us-east-1"} //nolint:exhaustruct
			server, err := zen.New(zen.Config{StreamRequestBody: true})                                            //nolint:exhaustruct
			require.NoError(t, err)
			server.RegisterRoute(nil, zen.NewRoute(http.MethodPost, "/", func(ctx context.Context, sess *zen.Session) error {
				start := svc.clock.Now()
				director := svc.makeInstanceDirector(sess, start)
				if destination == "region" {
					director = svc.makeRegionDirector(sess, start, "signed-metadata")
				}
				return svc.forward(ctx, sess, forwardConfig{
					targetURL: target, startTime: start, directorFunc: director,
					destination: destination, transport: http.DefaultTransport,
				})
			}))
			frontline := httptest.NewServer(server.Mux())
			t.Cleanup(frontline.Close)
			req, err := http.NewRequest(http.MethodPost, frontline.URL+requestURI, bytes.NewReader(body))
			require.NoError(t, err)
			req.Host = "customer.example"
			req.Header.Set("Content-Type", "application/octet-stream")
			response, err := frontline.Client().Do(req)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusCreated, response.StatusCode)
			require.Equal(t, "preserved", response.Header.Get("X-Customer-Response"))
			got := <-received
			require.NoError(t, got.err)
			require.Equal(t, requestURI, got.uri)
			require.Equal(t, "customer.example", got.host)
			require.Equal(t, http.MethodPost, got.method)
			require.Equal(t, "application/octet-stream", got.contentType)
			require.Equal(t, body, got.body)
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
