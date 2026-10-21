package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	handler "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRequestID_PreservedAcrossRegions(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		accept string
	}{
		{"success", http.StatusOK, "application/json"},
		{"JSON routing error", http.StatusServiceUnavailable, "application/json"},
		{"HTML routing error", http.StatusServiceUnavailable, "text/html"},
	} {
		t.Run(test.name, func(t *testing.T) {
			spans := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
			previous := tracing.GetGlobalTraceProvider()
			tracing.SetGlobalTraceProvider(provider)
			t.Cleanup(func() {
				tracing.SetGlobalTraceProvider(previous)
				require.NoError(t, provider.Shutdown(context.Background()))
			})

			rows := make(chan schema.FrontlineRequest, 3)
			logs := batch.New(batch.Config[schema.FrontlineRequest]{
				Name:          uid.New(uid.TestPrefix),
				BatchSize:     1,
				BufferSize:    3,
				Consumers:     1,
				FlushInterval: time.Hour,
				Flush: func(_ context.Context, records []schema.FrontlineRequest) {
					for _, record := range records {
						rows <- record
					}
				},
			})
			t.Cleanup(logs.Close)
			workloadHeaders := make(chan http.Header, 1)
			spoofedID := uid.New(uid.RequestPrefix)
			workload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				workloadHeaders <- r.Header.Clone()
				w.Header().Add(proxy.HeaderRequestID, spoofedID)
				w.Header().Add(proxy.HeaderRequestID, spoofedID)
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(workload.Close)
			codec, err := meta.New(testMetadataSigningKey)
			require.NoError(t, err)
			initialIDs := make(chan string, 3)
			regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
			var peer *httptest.Server
			for hop := len(regions) - 1; hop >= 0; hop-- {
				decision := router.RouteDecision{
					Destination:      router.DestinationLocalInstance,
					DeploymentID:     uid.New(uid.DeploymentPrefix),
					UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
				}
				if test.status == http.StatusOK {
					decision.LocalInstances = []db.FindInstancesByDeploymentIDRow{{
						ID:      uid.New(uid.InstancePrefix),
						Address: strings.TrimPrefix(workload.URL, "http://"),
					}}
				}
				var transport *http.Transport
				if peer != nil {
					decision.Destination = router.DestinationRemoteRegion
					decision.RemoteRegionAddress = regions[hop+1] + ".aws"
					peerURL, err := url.Parse(peer.URL)
					require.NoError(t, err)
					peerTransport, ok := peer.Client().Transport.(*http.Transport)
					require.True(t, ok)
					transport = peerTransport.Clone()
					transport.TLSClientConfig.ServerName = peerURL.Hostname()
					transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
						return (&net.Dialer{}).DialContext(ctx, network, peerURL.Host)
					}
					t.Cleanup(transport.CloseIdleConnections)
				}
				clk := clock.New()
				transports := proxy.NewTransportRegistry()
				t.Cleanup(transports.CloseIdleConnections)
				frontlineID := uid.New(uid.FrontlinePrefix)
				service, err := proxy.New(proxy.Config{
					InstanceID:         frontlineID,
					Platform:           "aws",
					Region:             regions[hop],
					ApexDomain:         "example.com",
					Clock:              clk,
					Metadata:           codec,
					UpstreamTransports: transports,
					Transport:          transport,
				})
				require.NoError(t, err)
				server, err := zen.New(zen.Config{StreamRequestBody: true})
				require.NoError(t, err)
				server.RegisterRoute([]zen.Middleware{
					middleware.WithReservedHeaderStrip(),
					func(next zen.HandleFunc) zen.HandleFunc {
						return func(ctx context.Context, sess *zen.Session) error {
							initialIDs <- sess.RequestID()
							return next(ctx, sess)
						}
					},
					middleware.WithClickHouseLogging(logs, clk, frontlineID, regions[hop], "aws"),
					middleware.WithObservability(errorpage.NewRenderer()),
				}, &handler.Handler{
					RouterService: &stubRouter{decision: decision},
					ProxyService:  service,
					Clock:         clk,
					Metadata:      codec,
				})
				peer = httptest.NewTLSServer(server.Mux())
				t.Cleanup(peer.Close)
			}

			req, err := http.NewRequest(http.MethodGet, peer.URL, nil)
			require.NoError(t, err)
			req.Header.Set("Accept", test.accept)
			req.Header.Set(proxy.HeaderRequestID, spoofedID)
			client := peer.Client()
			client.Timeout = 5 * time.Second
			response, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, test.status, response.StatusCode)
			require.Len(t, initialIDs, 3)
			ingressID := <-initialIDs
			require.NotEqual(t, spoofedID, ingressID)
			require.Equal(t, []string{ingressID}, response.Header.Values(proxy.HeaderRequestID))
			if test.status == http.StatusOK {
				require.Len(t, workloadHeaders, 1)
				headers := <-workloadHeaders
				require.Equal(t, []string{ingressID}, headers.Values(proxy.HeaderRequestID))
				require.Empty(t, headers.Values(proxy.HeaderFrontlineMeta))
				select {
				case row := <-rows:
					require.Equal(t, ingressID, row.RequestID)
				case <-time.After(5 * time.Second):
					t.Fatal("request log was not emitted")
				}
			} else if test.accept == "application/json" {
				var errorResponse middleware.ErrorResponse
				require.NoError(t, json.Unmarshal(body, &errorResponse))
				require.Equal(t, ingressID, errorResponse.Meta.RequestID)
			} else {
				require.Contains(t, string(body), ingressID)
			}
			require.Eventually(t, func() bool { return len(spans.Ended()) == 3 }, 5*time.Second, time.Millisecond)
			for _, span := range spans.Ended() {
				require.Contains(t, span.Attributes(), attribute.String("request_id", ingressID))
			}
		})
	}
}
