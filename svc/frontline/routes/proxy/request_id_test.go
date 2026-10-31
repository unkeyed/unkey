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
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
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
		name     string
		status   int
		accept   string
		fallback bool
	}{
		{"success", http.StatusOK, "application/json", false},
		{"JSON routing error", http.StatusServiceUnavailable, "application/json", false},
		{"HTML routing error", http.StatusServiceUnavailable, "text/html", false},
		{"local fallback success", http.StatusOK, "application/json", true},
		{"local fallback routing error", http.StatusServiceUnavailable, "application/json", true},
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
			var servers []*httptest.Server
			route := db.FindFrontlineRouteByFQDNRow{
				DeploymentID: uid.New(uid.DeploymentPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
				WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix), AppID: uid.New(uid.AppPrefix),
				DesiredState: db.DeploymentsDesiredStateRunning, UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
			}
			for hop := len(regions) - 1; hop >= 0; hop-- {
				var instances []db.FindInstancesByDeploymentIDRow
				if test.status == http.StatusOK {
					instances = []db.FindInstancesByDeploymentIDRow{{
						ID:      uid.New(uid.InstancePrefix),
						Address: strings.TrimPrefix(workload.URL, "http://"),
						Status:  db.InstancesStatusRunning, RegionName: regions[hop], RegionPlatform: "aws",
					}}
				}
				var transport *http.Transport
				if peer != nil {
					instances = []db.FindInstancesByDeploymentIDRow{{
						ID: uid.New(uid.InstancePrefix), Status: db.InstancesStatusRunning,
						RegionName: regions[hop+1], RegionPlatform: "aws",
					}}
					if test.fallback {
						instances = append(instances, db.FindInstancesByDeploymentIDRow{
							ID: uid.New(uid.InstancePrefix), Address: unreachableAddr(t), Status: db.InstancesStatusRunning,
							RegionName: regions[hop], RegionPlatform: "aws",
						})
					}
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
				routes, err := router.New(router.Config{
					Platform: "aws", Region: regions[hop], DB: regionQuerier{route: route, instances: instances},
					FrontlineRouteCache:   regionCache[db.FindFrontlineRouteByFQDNRow](t, "routes"),
					InstancesByDeployment: regionCache[[]db.FindInstancesByDeploymentIDRow](t, "instances"),
					PolicyCache:           regionCache[[]*frontlinev1.Policy](t, "policies"),
				})
				require.NoError(t, err)
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
					RouterService: routes,
					ProxyService:  service,
					Clock:         clk,
					Metadata:      codec,
				})
				peer = httptest.NewTLSServer(server.Mux())
				servers = append(servers, peer)
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
			for _, server := range servers {
				server.Close()
			}
			logs.Close()
			require.Len(t, rows, 1, "only the serving peer logs, including routing errors and local fallbacks")
			row := <-rows
			require.Equal(t, ingressID, row.RequestID)
			require.EqualValues(t, test.status, row.ResponseStatus)
			require.Equal(t, "eu-west-1", row.Region)
			require.Equal(t, route.WorkspaceID, row.WorkspaceID)
			require.Equal(t, route.ProjectID, row.ProjectID)
			require.Equal(t, route.AppID, row.AppID)
			require.Equal(t, route.EnvironmentID, row.EnvironmentID)
			require.Equal(t, route.DeploymentID, row.DeploymentID)
			require.Empty(t, row.RequestHeaders)
			require.Empty(t, row.ResponseHeaders)
			require.Empty(t, row.RequestBody)
			require.Empty(t, row.ResponseBody)
		})
	}
}
