package handler_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
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
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
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
	clock  *clock.TestClock
	code   codes.URN
	result policies.Result
}

func (e rejectingEvaluator) Evaluate(context.Context, *zen.Session, *http.Request, string, string, []*frontlinev1.Policy) (policies.Result, error) {
	e.clock.Tick(37 * time.Millisecond)
	return e.result, fault.New("policy rejected request", fault.Code(e.code))
}

func TestRequestLogging_KeyAuthRejectionCapture(t *testing.T) {
	for _, tc := range []struct {
		name        string
		code        codes.URN
		status      int
		accept      string
		logging     *frontlinev1.Logging
		beforeAuth  bool
		disabled    bool
		path        string
		redact      bool
		invalidSpec bool
	}{
		{name: "missing_key_logging_first", code: codes.Frontline.Auth.MissingCredentials.URN(), status: http.StatusUnauthorized, accept: "application/json", logging: allCaptureFlags(), beforeAuth: true},
		{name: "invalid_key_logging_last", code: codes.Frontline.Auth.InvalidKey.URN(), status: http.StatusUnauthorized, accept: "application/json", logging: allCaptureFlags()},
		{name: "forbidden_json", code: codes.Frontline.Auth.InsufficientPermissions.URN(), status: http.StatusForbidden, accept: "application/json", logging: allCaptureFlags()},
		{name: "forbidden_html", code: codes.Frontline.Auth.InsufficientPermissions.URN(), status: http.StatusForbidden, accept: "text/html", logging: allCaptureFlags()},
		{name: "disabled", code: codes.Frontline.Auth.InvalidKey.URN(), status: http.StatusUnauthorized, logging: allCaptureFlags(), disabled: true},
		{name: "nonmatching", code: codes.Frontline.Auth.InvalidKey.URN(), status: http.StatusUnauthorized, logging: allCaptureFlags(), path: "/other"},
		{name: "headers_only", code: codes.Frontline.Auth.InvalidKey.URN(), status: http.StatusUnauthorized, logging: &frontlinev1.Logging{RequestHeaders: true, ResponseHeaders: true}},
		{name: "request_body_only", code: codes.Frontline.Auth.InvalidKey.URN(), status: http.StatusUnauthorized, logging: &frontlinev1.Logging{RequestBody: true}},
		{name: "response_body_only", code: codes.Frontline.Auth.InvalidKey.URN(), status: http.StatusUnauthorized, logging: &frontlinev1.Logging{ResponseBody: true}},
		{name: "redaction_after_auth", code: codes.Frontline.Auth.InsufficientPermissions.URN(), status: http.StatusForbidden, accept: "application/json", logging: allCaptureFlags(), redact: true},
		{name: "invalid_redaction_spec", code: codes.Frontline.Auth.InsufficientPermissions.URN(), status: http.StatusForbidden, accept: "application/json", logging: allCaptureFlags(), invalidSpec: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rows []schema.FrontlineRequest
			buf := batch.New(batch.Config[schema.FrontlineRequest]{
				Name: "test", BatchSize: 10, BufferSize: 10, FlushInterval: time.Hour, Consumers: 1,
				Flush: func(_ context.Context, batch []schema.FrontlineRequest) { rows = append(rows, batch...) },
			})
			t.Cleanup(buf.Close)
			clk := clock.NewTestClock()
			limiter := ratelimit.NewLocal(clk)
			t.Cleanup(func() { require.NoError(t, limiter.Close()) })
			engine, err := policies.New(policies.Config{
				KeyAuth: rejectingAuthenticator{code: tc.code}, RateLimiter: limiter, Clock: clk,
				OpenapiBodyBytesMax: 10 << 20, OpenapiBodyBytesBudget: 512 << 20,
			})
			require.NoError(t, err)
			t.Cleanup(engine.Close)
			logging := &frontlinev1.Policy{
				Enabled: new(!tc.disabled), Config: &frontlinev1.Policy_Logging{Logging: tc.logging},
			}
			if tc.path != "" {
				logging.Match = []*frontlinev1.MatchExpr{{Expr: &frontlinev1.MatchExpr_Path{Path: &frontlinev1.PathMatch{
					Path: &frontlinev1.StringMatch{Match: &frontlinev1.StringMatch_Exact{Exact: tc.path}},
				}}}}
			}
			auth := &frontlinev1.Policy{Enabled: new(true), Config: &frontlinev1.Policy_Keyauth{Keyauth: &frontlinev1.KeyAuth{
				Locations: []*frontlinev1.KeyLocation{
					{Location: &frontlinev1.KeyLocation_Header{Header: &frontlinev1.HeaderKeyLocation{Name: "X-Api-Key"}}},
					{Location: &frontlinev1.KeyLocation_QueryParam{QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "api_key"}}},
				},
			}}}
			decision := localDecision("upstream.test")
			decision.Policies = []*frontlinev1.Policy{auth, logging}
			if tc.beforeAuth {
				decision.Policies = []*frontlinev1.Policy{logging, auth}
			}
			if tc.redact || tc.invalidSpec {
				spec := rejectionRedactionSpec
				if tc.invalidSpec {
					spec = "not valid yaml: [[["
				}
				decision.Policies = append(decision.Policies, &frontlinev1.Policy{
					Enabled: new(true), Config: &frontlinev1.Policy_Openapi{Openapi: &frontlinev1.OpenApiRequestValidation{SpecYaml: []byte(spec)}},
				})
			}
			upstream := &captureRequestProxy{}
			h := handler.Handler{RouterService: &stubRouter{decision: decision}, ProxyService: upstream, Engine: engine, Clock: clk}
			serve := middleware.WithClickHouseLogging(buf, clk, uid.New(uid.FrontlinePrefix), "us-east-1", "test")(
				middleware.WithObservability(errorpage.NewRenderer())(zen.WithTimeout(time.Minute)(h.Handle)),
			)
			payload, err := json.Marshal(map[string]string{"secret": "hide-me", "visible": "keep-me"})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "https://example.test/orders?api_key=secret&view=full", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer secret")
			req.Header.Set("X-Api-Key", "secret")
			req.Header.Set("User-Agent", "capture-test/1.0")
			req.Header.Set("Accept", tc.accept)
			recorder := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(recorder, req, 0))
			require.NoError(t, serve(t.Context(), sess))
			buf.Close()

			require.Equal(t, tc.status, recorder.Code)
			require.Nil(t, upstream.received)
			require.Len(t, rows, 1)
			row := rows[0]
			require.Equal(t, decision.DeploymentID, row.DeploymentID)
			require.EqualValues(t, tc.status, row.ResponseStatus)
			enabled := !tc.disabled && tc.path == ""
			if enabled && tc.logging.GetRequestHeaders() {
				require.Contains(t, row.RequestHeaders, "Authorization: [REDACTED]")
				require.Contains(t, row.RequestHeaders, "X-Api-Key: [REDACTED]")
				require.Equal(t, "capture-test/1.0", row.UserAgent)
			} else {
				require.Empty(t, row.RequestHeaders)
				require.Empty(t, row.UserAgent)
			}
			if enabled && tc.logging.GetResponseHeaders() {
				require.Contains(t, row.ResponseHeaders, "Content-Type: "+recorder.Header().Get("Content-Type"))
			} else {
				require.Empty(t, row.ResponseHeaders)
			}
			if enabled && tc.logging.GetQuery() {
				require.Equal(t, []string{"[REDACTED]"}, row.QueryParams["api_key"])
				require.Equal(t, []string{"full"}, row.QueryParams["view"])
				require.NotContains(t, row.QueryString, "secret")
			} else {
				require.Empty(t, row.QueryParams)
				require.Empty(t, row.QueryString)
			}
			if enabled && tc.logging.GetRequestBody() && !tc.invalidSpec {
				var logged map[string]string
				require.NoError(t, json.Unmarshal([]byte(row.RequestBody), &logged))
				secret := "hide-me"
				if tc.redact {
					secret = "[REDACTED]"
				}
				require.Equal(t, map[string]string{"secret": secret, "visible": "keep-me"}, logged)
			} else {
				require.Empty(t, row.RequestBody)
			}
			if enabled && tc.logging.GetResponseBody() && !tc.invalidSpec {
				require.Equal(t, recorder.Body.String(), row.ResponseBody)
				if tc.accept == "application/json" {
					var response middleware.ErrorResponse
					require.NoError(t, json.Unmarshal([]byte(row.ResponseBody), &response))
					require.Equal(t, string(tc.code), response.Error.Code)
					require.Equal(t, sess.RequestID(), response.Meta.RequestID)
				}
			} else {
				require.Empty(t, row.ResponseBody)
			}
		})
	}
}

const rejectionRedactionSpec = `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /orders:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [missing]
              properties:
                secret:
                  type: string
                  x-unkey-redact: true
                visible:
                  type: string
      responses:
        "200":
          description: ok
`

func TestRequestLogging_RejectedUploadIsBounded(t *testing.T) {
	for _, tc := range []struct {
		name          string
		payload       string
		contentLength int
	}{
		{name: "oversized", payload: strings.Repeat("x", zen.MaxBodyCapture+19), contentLength: zen.MaxBodyCapture + 19},
		{name: "stalled", payload: "partial upload", contentLength: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rows []schema.FrontlineRequest
			buf := batch.New(batch.Config[schema.FrontlineRequest]{
				Name: "test", BatchSize: 10, BufferSize: 10, FlushInterval: time.Hour, Consumers: 1,
				Flush: func(_ context.Context, batch []schema.FrontlineRequest) { rows = append(rows, batch...) },
			})
			t.Cleanup(buf.Close)
			clk := clock.NewTestClock()
			decision := localDecision("upstream.test")
			decision.Policies = []*frontlinev1.Policy{{}}
			upstream := &captureRequestProxy{}
			h := handler.Handler{
				RouterService: &stubRouter{decision: decision}, ProxyService: upstream, Clock: clk,
				Engine: rejectingEvaluator{clock: clk, code: codes.Frontline.Auth.InsufficientPermissions.URN(), result: policies.Result{LogRequestBody: true, LogResponseBody: true}},
			}
			frontline, err := zen.New(zen.Config{StreamRequestBody: true, ReadTimeout: -1, WriteTimeout: -1})
			require.NoError(t, err)
			frontline.RegisterRoute([]zen.Middleware{
				middleware.WithClickHouseLogging(buf, clk, uid.New(uid.FrontlinePrefix), "us-east-1", "test"),
				middleware.WithObservability(errorpage.NewRenderer()),
				zen.WithTimeout(time.Minute),
			}, &h)
			server := httptest.NewServer(frontline)
			t.Cleanup(server.Close)
			conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })
			require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
			_, err = fmt.Fprintf(conn, "POST /orders HTTP/1.1\r\nHost: example.test\r\nAccept: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", tc.contentLength, tc.payload)
			require.NoError(t, err)
			response, err := http.ReadResponse(bufio.NewReader(conn), nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			server.Close()
			buf.Close()

			require.Equal(t, http.StatusForbidden, response.StatusCode)
			require.Nil(t, upstream.received)
			require.Len(t, rows, 1)
			require.EqualValues(t, http.StatusForbidden, rows[0].ResponseStatus)
			require.Equal(t, tc.payload[:min(len(tc.payload), zen.MaxBodyCapture)], rows[0].RequestBody)
			require.Equal(t, string(body), rows[0].ResponseBody)
		})
	}
}

func allCaptureFlags() *frontlinev1.Logging {
	return &frontlinev1.Logging{RequestHeaders: true, ResponseHeaders: true, RequestBody: true, ResponseBody: true, Query: true}
}

type rejectingAuthenticator struct{ code codes.URN }

func (a rejectingAuthenticator) Execute(context.Context, *zen.Session, *http.Request, string, *frontlinev1.KeyAuth) (*principal.Principal, error) {
	return nil, fault.New("authentication rejected request", fault.Code(a.code))
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
