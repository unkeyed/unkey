package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/paseto"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/metrics"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
)

func TestRegionFallback_ExhaustedInstancesReturnUpstreamError(t *testing.T) {
	for _, accept := range []string{"application/json", "text/html"} {
		t.Run(accept, func(t *testing.T) {
			platform := metrics.RequestsTotal.WithLabelValues("5xx", "platform", string(codes.Frontline.Internal.InternalServerError.URN()))
			upstream := metrics.RequestsTotal.WithLabelValues("5xx", "upstream", string(codes.Frontline.Proxy.BadGateway.URN()))
			platformBefore := requestCount(t, platform)
			upstreamBefore := requestCount(t, upstream)
			regions := startRegions(t, []string{"us-east-1", "us-west-2"}, map[string]string{
				"us-east-1": unreachableAddr(t),
				"us-west-2": unreachableAddr(t),
			})
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+regions["us-east-1"]+"/orders", strings.NewReader("create-order"))
			require.NoError(t, err)
			req.Host = "retry-test.example.com"
			req.Header.Set("Accept", accept)
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, http.StatusBadGateway, resp.StatusCode, "body: %s", body)
			require.Equal(t, []string{"aws::us-east-1", "aws::us-west-2"}, resp.Header.Values(proxy.HeaderRegion))
			if accept == "application/json" {
				var result middleware.ErrorResponse
				require.NoError(t, json.Unmarshal(body, &result))
				require.Equal(t, string(codes.Frontline.Proxy.BadGateway.URN()), result.Error.Code)
				require.Contains(t, result.Error.Message, "refused the connection")
				require.Contains(t, result.Error.Message, "configured port")
			} else {
				require.Contains(t, string(body), string(codes.Frontline.Proxy.BadGateway.URN()))
				require.Contains(t, string(body), "refused the connection")
				require.Contains(t, string(body), "configured port")
			}
			require.NotContains(t, string(body), "127.0.0.1")
			require.Eventually(t, func() bool {
				return requestCount(t, upstream) == upstreamBefore+1
			}, time.Second, time.Millisecond)
			require.Equal(t, platformBefore, requestCount(t, platform))
		})
	}
}

func TestRegionFallback_ExcludesSignedRoutingRegion(t *testing.T) {
	for _, hop := range []meta.Hop{
		{Region: "display label", RegionPlatform: "us-east-1.aws"},
		{Region: "aws::eu-north-1", RegionPlatform: "us-east-1.aws"},
		{Region: "aws::us-east-1"},
	} {
		t.Run(hop.Region, func(t *testing.T) {
			backend, stop := startBackend(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			t.Cleanup(stop)
			regions := startRegions(t, []string{"us-east-1", "us-west-2"}, map[string]string{
				"us-east-1": backend,
			})
			codec, err := meta.New(testMetadataSigningKey)
			require.NoError(t, err)
			token, err := codec.Marshal(&meta.Metadata{
				Claims: paseto.Claims{ExpiresAt: time.Now().Add(time.Minute)},
				Hops:   []meta.Hop{hop},
			})
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+regions["us-west-2"], nil)
			require.NoError(t, err)
			req.Host = "retry-test.example.com"
			req.Header.Set(proxy.HeaderFrontlineMeta, token)
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "body: %s", body)
			require.Empty(t, resp.Header.Values(proxy.HeaderRegion))
		})
	}
}

func TestRegionFallback_ReachesUnvisitedRegionWithPostBody(t *testing.T) {
	for _, entry := range []string{"us-east-1", "eu-west-1"} {
		t.Run(entry, func(t *testing.T) {
			backend, stop := startBackend(t, func(w http.ResponseWriter, req *http.Request) {
				_, err := io.Copy(w, req.Body)
				if err != nil {
					t.Error(err)
				}
			})
			t.Cleanup(stop)
			regions := startRegions(t, []string{"us-east-1", "us-west-2", "eu-north-1", "eu-west-1"}, map[string]string{
				"us-east-1":  unreachableAddr(t),
				"us-west-2":  unreachableAddr(t),
				"eu-north-1": backend,
			})
			payload := "order-with-non-replayable-body"
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+regions[entry]+"/orders", strings.NewReader(payload))
			require.NoError(t, err)
			req.Host = "retry-test.example.com"
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", body)
			require.Equal(t, payload, string(body))
			expected := []string{"aws::us-east-1", "aws::us-west-2", "aws::eu-north-1"}
			if entry == "eu-west-1" {
				expected = []string{"aws::eu-west-1", "aws::eu-north-1"}
			}
			require.Equal(t, expected, resp.Header.Values(proxy.HeaderRegion))
		})
	}
}

func startRegions(t *testing.T, names []string, backends map[string]string) map[string]string {
	t.Helper()
	workspaceID := uid.New(uid.WorkspacePrefix)
	projectID := uid.New(uid.ProjectPrefix)
	appID := uid.New(uid.AppPrefix)
	instances := make([]db.FindInstancesByDeploymentIDRow, 0, len(backends))
	for region, address := range backends {
		instances = append(instances, db.FindInstancesByDeploymentIDRow{
			ID: uid.New(uid.InstancePrefix), Address: address, Status: db.InstancesStatusRunning,
			RegionName: region, RegionPlatform: "aws",
			WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID,
		})
	}
	queries := regionQuerier{
		route: db.FindFrontlineRouteByFQDNRow{
			DeploymentID: uid.New(uid.DeploymentPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
			DesiredState: db.DeploymentsDesiredStateRunning, UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
		},
		instances: instances,
	}
	metadata, err := meta.New(testMetadataSigningKey)
	require.NoError(t, err)
	addresses := make(map[string]string, len(names))
	peers := &regionTransport{addresses: addresses, transport: &http.Transport{}}
	t.Cleanup(peers.transport.CloseIdleConnections)
	for _, region := range names {
		routes, err := router.New(router.Config{
			Platform: "aws", Region: region, DB: queries,
			FrontlineRouteCache:   regionCache[db.FindFrontlineRouteByFQDNRow](t, "frontline_routes"),
			InstancesByDeployment: regionCache[[]db.FindInstancesByDeploymentIDRow](t, "instances"),
			PolicyCache:           regionCache[[]*frontlinev1.Policy](t, "policies"),
		})
		require.NoError(t, err)
		transport := &http.Transport{}
		t.Cleanup(transport.CloseIdleConnections)
		transport.RegisterProtocol("https", peers)
		upstreams := proxy.NewTransportRegistry()
		t.Cleanup(upstreams.CloseIdleConnections)
		service, err := proxy.New(proxy.Config{
			InstanceID: uid.New(uid.FrontlinePrefix), Platform: "aws", Region: region,
			ApexDomain: "test.local", Clock: clock.New(), MaxHops: 10, Metadata: metadata,
			Transport: transport, UpstreamTransports: upstreams,
		})
		require.NoError(t, err)
		address, stop := startFrontlineWithRouter(t, routes, service, false)
		t.Cleanup(stop)
		peers.mu.Lock()
		addresses[region] = address
		peers.mu.Unlock()
	}
	return addresses
}

type regionQuerier struct {
	db.Querier
	route     db.FindFrontlineRouteByFQDNRow
	instances []db.FindInstancesByDeploymentIDRow
}

func (q regionQuerier) FindFrontlineRouteByFQDN(context.Context, string) (db.FindFrontlineRouteByFQDNRow, error) {
	return q.route, nil
}

func (q regionQuerier) FindInstancesByDeploymentID(context.Context, string) ([]db.FindInstancesByDeploymentIDRow, error) {
	return q.instances, nil
}

func regionCache[V any](t *testing.T, resource string) cache.Cache[string, V] {
	t.Helper()
	c, err := cache.New(cache.Config[string, V]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 10, Resource: resource, Clock: clock.New(),
	})
	require.NoError(t, err)
	return c
}

type regionTransport struct {
	mu        sync.RWMutex
	addresses map[string]string
	transport *http.Transport
}

func (r *regionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	region := strings.TrimSuffix(strings.TrimPrefix(req.URL.Host, "frontline."), ".aws.test.local")
	r.mu.RLock()
	address, ok := r.addresses[region]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown test region %q", region)
	}
	out := req.Clone(req.Context())
	out.URL.Scheme = "http"
	out.URL.Host = address
	return r.transport.RoundTrip(out)
}

func requestCount(t *testing.T, counter prometheus.Counter) float64 {
	t.Helper()
	var value dto.Metric
	require.NoError(t, counter.Write(&value))
	return value.GetCounter().GetValue()
}
