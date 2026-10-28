package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/metrics"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/routes"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestOperationalLoggingKeepsFaultsAndCustomerTelemetry(t *testing.T) {
	logs := loggertest.Install(t)
	spans := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
	previous := tracing.GetGlobalTraceProvider()
	tracing.SetGlobalTraceProvider(provider)
	t.Cleanup(func() {
		tracing.SetGlobalTraceProvider(previous)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	for _, tc := range []struct {
		name        string
		code        codes.URN
		category    codes.Category
		status      int
		operational bool
		unresolved  bool
		canceled    bool
	}{
		{name: "success", status: 200},
		{name: "app_bad_request", status: 400},
		{name: "app_internal_error", status: 500},
		{name: "invalid_input", code: codes.App.Validation.InvalidInput.URN(), status: 400},
		{name: "body_too_large", code: codes.User.BadRequest.RequestBodyTooLarge.URN(), status: 413},
		{name: "body_unreadable", code: codes.User.BadRequest.RequestBodyUnreadable.URN(), status: 400},
		{name: "missing_auth", code: codes.Auth.Authentication.Missing.URN(), status: 401},
		{name: "malformed_auth", code: codes.Auth.Authentication.Malformed.URN(), status: 401},
		{name: "missing_credentials", code: codes.Frontline.Auth.MissingCredentials.URN(), status: 401},
		{name: "invalid_key", code: codes.Frontline.Auth.InvalidKey.URN(), status: 401},
		{name: "permissions", code: codes.Frontline.Auth.InsufficientPermissions.URN(), status: 403},
		{name: "rate_limit", code: codes.Frontline.Auth.RateLimited.URN(), status: 429},
		{name: "credits", code: codes.Frontline.Auth.UsageExceeded.URN(), status: 429},
		{name: "firewall", code: codes.Frontline.Firewall.Denied.URN(), status: 403},
		{name: "openapi", code: codes.Frontline.OpenApi.InvalidRequest.URN(), status: 400},
		{name: "budget", code: codes.Frontline.Routing.SpendLimitReached.URN(), status: 402},
		{name: "offline", code: codes.Frontline.Routing.DeploymentOffline.URN(), status: 503},
		{name: "invalid_config", code: codes.Frontline.Internal.InvalidConfiguration.URN(), status: 422},
		{name: "missing_rate_limit", code: codes.Frontline.Internal.RatelimitNotConfigured.URN(), status: 422},
		{name: "client_disconnect", code: codes.User.BadRequest.ClientClosedRequest.URN(), status: 499, canceled: true},
		{name: "unexpected_cancellation", code: codes.User.BadRequest.ClientClosedRequest.URN(), status: 499, operational: true},
		{name: "no_instances", code: codes.Frontline.Routing.NoRunningInstances.URN(), status: 503, operational: true},
		{name: "unknown_host", code: codes.Frontline.Routing.ConfigNotFound.URN(), status: 404, operational: true, unresolved: true},
		{name: "database", code: codes.Frontline.Internal.ConfigLoadFailed.URN(), status: 500, operational: true, unresolved: true},
		{name: "proxy_fault", code: codes.Frontline.Proxy.BadGateway.URN(), status: 502, operational: true},
		{name: "platform_override", code: codes.Frontline.Auth.InvalidKey.URN(), category: codes.CategoryPlatform, status: 401, operational: true},
		{name: "timeout", code: codes.User.BadRequest.RequestTimeout.URN(), status: 504, operational: true},
		{name: "untagged", status: 500, operational: true, unresolved: true},
		{name: "unknown_code", code: "err:frontline:client:new_error", status: 500, operational: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}))
			t.Cleanup(app.Close)
			decision := router.RouteDecision{
				Destination: router.DestinationLocalInstance, UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
				WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix),
				AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix), DeploymentID: uid.New(uid.DeploymentPrefix),
				LocalInstances: []db.FindInstancesByDeploymentIDRow{{ID: uid.New(uid.InstancePrefix), Address: strings.TrimPrefix(app.URL, "http://")}},
			}
			if tc.unresolved {
				decision = router.RouteDecision{}
			}
			var routeErr error
			if tc.code != "" {
				routeErr = fault.New("request failed", fault.Code(tc.code), fault.Category(tc.category))
			} else if tc.operational {
				routeErr = errors.New("database unavailable")
			}
			var rows []schema.FrontlineRequest
			analytics := batch.New(batch.Config[schema.FrontlineRequest]{
				Name: "test", BatchSize: 10, BufferSize: 10, Consumers: 1, FlushInterval: time.Hour,
				Flush: func(_ context.Context, batch []schema.FrontlineRequest) { rows = append(rows, batch...) },
			})
			t.Cleanup(analytics.Close)
			clk := clock.New()
			metadata, err := meta.New(testMetadataSigningKey)
			require.NoError(t, err)
			transports := proxy.NewTransportRegistry()
			t.Cleanup(transports.CloseIdleConnections)
			upstream, err := proxy.New(proxy.Config{Clock: clk, Metadata: metadata, UpstreamTransports: transports})
			require.NoError(t, err)
			server, err := zen.New(zen.Config{StreamRequestBody: true})
			require.NoError(t, err)
			routes.Register(server, &routes.Services{
				RouterService: &operationalRouter{decision: decision, err: routeErr}, ProxyService: upstream,
				Clock: clk, Metadata: metadata, FrontlineRequests: analytics, ErrorPageRenderer: errorpage.NewRenderer(),
			})
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if tc.canceled {
				cancel()
			}
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/orders?api_key=secret", nil)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Authorization", "Bearer secret")
			response := httptest.NewRecorder()
			domain := ""
			code := tc.code
			if routeErr != nil {
				category, ok := fault.GetCategory(routeErr)
				domain = string(category)
				if !ok {
					domain = "platform"
					code = codes.Frontline.Internal.InternalServerError.URN()
				}
			}
			counter := metrics.RequestsTotal.WithLabelValues(metrics.StatusClass(tc.status), domain, string(code))
			before := requestCount(t, counter)
			snapshot := logs.Snapshot()
			server.Mux().ServeHTTP(response, req)
			analytics.Close()
			require.Equal(t, tc.status, response.Code)
			requestID := response.Header().Get(proxy.HeaderRequestID)
			require.NotEmpty(t, requestID)
			records := logs.Since(snapshot)
			if tc.operational {
				require.Len(t, records, 1)
				attrs := loggertest.FlatAttrs(records[0])
				require.Equal(t, "frontline.request_error", attrs["event"])
				require.Equal(t, "frontline", attrs["responseSource"])
				require.Equal(t, requestID, attrs["requestId"])
				require.NotEmpty(t, attrs["error"])
				require.Equal(t, string(code), attrs["code"])
			} else {
				require.Empty(t, records)
			}
			if tc.unresolved {
				require.Empty(t, rows)
			} else {
				require.Len(t, rows, 1)
				require.Equal(t, requestID, rows[0].RequestID)
				require.Equal(t, decision.WorkspaceID, rows[0].WorkspaceID)
				require.EqualValues(t, tc.status, rows[0].ResponseStatus)
				require.Empty(t, rows[0].RequestHeaders)
				require.Empty(t, rows[0].QueryString)
			}
			require.Equal(t, before+1, requestCount(t, counter))
			ended := spans.Ended()
			require.NotEmpty(t, ended)
			require.Contains(t, ended[len(ended)-1].Attributes(), attribute.String("request_id", requestID))
			require.Contains(t, ended[len(ended)-1].Attributes(), attribute.Int("status_code", tc.status))
		})
	}
}

type operationalRouter struct {
	router.Service
	decision router.RouteDecision
	err      error
}

func (r *operationalRouter) Route(context.Context, string, ...string) (router.RouteDecision, error) {
	return r.decision, r.err
}
