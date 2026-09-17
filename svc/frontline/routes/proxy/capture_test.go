package handler_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	handler "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

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

// TestRejectedRequestKeepsCaptureFlags pins that a request a policy denies
// still carries the capture opt-ins of the logging policies that matched
// before the denial. Without them the ClickHouse row for the rejection would
// arrive without the headers that explain why it was rejected.
func TestRejectedRequestKeepsCaptureFlags(t *testing.T) {
	sess := &zen.Session{}
	req := httptest.NewRequest(http.MethodGet, "http://example.test", nil)
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	tracking := &proxy.RequestTracking{}
	ctx := proxy.WithRequestTracking(t.Context(), tracking)
	decision := localDecision("upstream.test")
	decision.Policies = []*frontlinev1.Policy{{}}
	upstream := &captureRequestProxy{}
	h := handler.Handler{
		RouterService: &stubRouter{decision: decision},
		ProxyService:  upstream,
		Engine:        rejectingEvaluator{},
		Clock:         clock.NewTestClock(),
	}

	err := h.Handle(ctx, sess)

	require.ErrorIs(t, err, errRejected)
	require.Nil(t, upstream.received, "a denied request must not reach an instance")
	require.Empty(t, tracking.InstanceID)
	require.True(t, tracking.LogRequestHeaders)
	require.True(t, tracking.LogQuery)
}

var errRejected = errors.New("denied")

// rejectingEvaluator mimics a logging policy matching before an enforcement
// policy denies the request: Evaluate returns the flags it already collected
// alongside the error.
type rejectingEvaluator struct{}

func (rejectingEvaluator) Evaluate(context.Context, *zen.Session, *http.Request, string, string, []*frontlinev1.Policy) (policies.Result, error) {
	//nolint:exhaustruct
	return policies.Result{LogRequestHeaders: true, LogQuery: true}, errRejected
}
