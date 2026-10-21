package handler_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	handler "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

func TestRequestLogging_PolicyRejections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   codes.URN
		status int
	}{
		{"missing_key", codes.Frontline.Auth.MissingCredentials.URN(), http.StatusUnauthorized},
		{"invalid_key", codes.Frontline.Auth.InvalidKey.URN(), http.StatusUnauthorized},
		{"permissions", codes.Frontline.Auth.InsufficientPermissions.URN(), http.StatusForbidden},
		{"rate_limit", codes.Frontline.Auth.RateLimited.URN(), http.StatusTooManyRequests},
		{"credits", codes.Frontline.Auth.UsageExceeded.URN(), http.StatusTooManyRequests},
		{"firewall", codes.Frontline.Firewall.Denied.URN(), http.StatusForbidden},
		{"validation", codes.Frontline.OpenApi.InvalidRequest.URN(), http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rows []schema.FrontlineRequest
			buf := batch.New(batch.Config[schema.FrontlineRequest]{
				Name:          "test",
				BatchSize:     10,
				BufferSize:    10,
				FlushInterval: time.Hour,
				Consumers:     1,
				Flush: func(_ context.Context, batch []schema.FrontlineRequest) {
					rows = append(rows, batch...)
				},
			})
			t.Cleanup(buf.Close)
			clk := clock.NewTestClock()
			decision := router.RouteDecision{
				Destination:   router.DestinationLocalInstance,
				DeploymentID:  uid.New(uid.DeploymentPrefix),
				WorkspaceID:   uid.New(uid.WorkspacePrefix),
				ProjectID:     uid.New(uid.ProjectPrefix),
				AppID:         uid.New(uid.AppPrefix),
				EnvironmentID: uid.New(uid.EnvironmentPrefix),
				LocalInstances: []db.FindInstancesByDeploymentIDRow{{
					ID:      uid.New(uid.InstancePrefix),
					Address: "upstream.test",
				}},
				Policies: []*frontlinev1.Policy{{}},
			}
			upstream := &captureRequestProxy{}
			h := handler.Handler{
				RouterService: &stubRouter{decision: decision},
				ProxyService:  upstream,
				Engine:        rejectingEvaluator{clock: clk, code: tc.code},
				Clock:         clk,
			}
			frontlineID := uid.New(uid.FrontlinePrefix)
			serve := middleware.WithClickHouseLogging(buf, clk, frontlineID, "us-east-1", "test")(
				middleware.WithObservability(errorpage.NewRenderer())(h.Handle),
			)
			req := httptest.NewRequest(http.MethodPost, "https://example.test/orders?secret=hidden", strings.NewReader("private body"))
			req.Header.Set("Authorization", "Bearer secret")
			req.Header.Set("Accept", "application/json")
			recorder := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(recorder, req, 0))
			require.NoError(t, serve(t.Context(), sess))
			buf.Close()

			require.Equal(t, tc.status, recorder.Code)
			require.Nil(t, upstream.received)
			require.Len(t, rows, 1)
			row := rows[0]
			require.Equal(t, sess.RequestID(), row.RequestID)
			require.Equal(t, decision.DeploymentID, row.DeploymentID)
			require.Equal(t, decision.WorkspaceID, row.WorkspaceID)
			require.Equal(t, decision.ProjectID, row.ProjectID)
			require.Equal(t, decision.AppID, row.AppID)
			require.Equal(t, decision.EnvironmentID, row.EnvironmentID)
			require.Equal(t, frontlineID, row.FrontlineID)
			require.Equal(t, "us-east-1", row.Region)
			require.Equal(t, http.MethodPost, row.Method)
			require.Equal(t, "example.test", row.Host)
			require.Equal(t, "/orders", row.Path)
			require.EqualValues(t, tc.status, row.ResponseStatus)
			require.Empty(t, row.InstanceID)
			require.Empty(t, row.InstanceAddress)
			require.Zero(t, row.InstanceLatency)
			require.EqualValues(t, 37, row.TotalLatency)
			require.EqualValues(t, 37, row.GatewayLatency)
			require.Empty(t, row.RequestHeaders)
			require.Empty(t, row.ResponseHeaders)
			require.Empty(t, row.QueryString)
			require.Empty(t, row.QueryParams)
			require.Empty(t, row.RequestBody)
			require.Empty(t, row.ResponseBody)
			require.Empty(t, row.UserAgent)
			require.Empty(t, row.IPAddress)
		})
	}
}

type rejectingEvaluator struct {
	clock *clock.TestClock
	code  codes.URN
}

func (e rejectingEvaluator) Evaluate(context.Context, *zen.Session, *http.Request, string, string, []*frontlinev1.Policy) (policies.Result, error) {
	e.clock.Tick(37 * time.Millisecond)
	return policies.Result{}, fault.New("policy rejected request", fault.Code(e.code))
}

func TestRequestCaptureIsOptInAndDoesNotTruncateUpstream(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%t", enabled), func(t *testing.T) {
			payload := strings.Repeat("x", zen.MaxBodyCapture+19)
			sess := &zen.Session{}
			req := httptest.NewRequest(http.MethodPost, "http://example.test", strings.NewReader(payload))
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			tracking := &proxy.RequestTracking{}
			ctx := proxy.WithRequestTracking(t.Context(), tracking)
			decision := localDecision("upstream.test")
			decision.Policies = []*frontlinev1.Policy{{}}
			upstream := &captureRequestProxy{}
			h := handler.Handler{
				RouterService: &stubRouter{decision: decision},
				ProxyService:  upstream,
				Engine:        captureRequestEvaluator{enabled: enabled},
				Clock:         clock.NewTestClock(),
			}
			require.NoError(t, h.Handle(ctx, sess))
			require.Equal(t, payload, string(upstream.received))
			if enabled {
				require.Equal(t, payload[:zen.MaxBodyCapture], string(tracking.RequestBody))
				require.LessOrEqual(t, cap(tracking.RequestBody), zen.MaxBodyCapture)
			} else {
				require.Nil(t, tracking.RequestBody)
			}
		})
	}
}

type captureRequestEvaluator struct{ enabled bool }

func (e captureRequestEvaluator) Evaluate(context.Context, *zen.Session, *http.Request, string, string, []*frontlinev1.Policy) (policies.Result, error) {
	return policies.Result{LogRequestBody: e.enabled}, nil
}

type captureRequestProxy struct {
	proxy.Service
	received []byte
}

func (p *captureRequestProxy) ForwardToInstance(_ context.Context, sess *zen.Session, _ db.DeploymentsUpstreamProtocol, _ db.FindInstancesByDeploymentIDRow) error {
	var err error
	p.received, err = io.ReadAll(sess.Request().Body)
	return err
}
