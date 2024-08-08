package testutil

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

type Harness struct {
	t *testing.T

	logger  logging.Logger
	metrics metrics.Metrics

	ratelimit ratelimit.Service

	api humatest.TestAPI
}

func NewHarness(t *testing.T) *Harness {
	_, api := humatest.New(t)

	nodeId := uid.New("test")
	authToken := uid.New("test")
	serfAddr := fmt.Sprintf("localhost:%d", port.Get())
	rpcAddr := fmt.Sprintf("localhost:%d", port.Get())

	h := Harness{
		t:       t,
		logger:  logging.NewNoopLogger(),
		metrics: metrics.NewNoop(),
		api:     api,
	}

	memb, err := membership.New(membership.Config{
		NodeId:   nodeId,
		SerfAddr: serfAddr,
	})
	require.NoError(t, err)

	c, err := cluster.New(cluster.Config{
		NodeId:     nodeId,
		Membership: memb,
		Logger:     h.logger,
		Metrics:    h.metrics,
		AuthToken:  authToken,
		RpcAddr:    rpcAddr,
	})
	require.NoError(t, err)
	rl, err := ratelimit.New(ratelimit.Config{
		Logger:  h.logger,
		Metrics: h.metrics,
		Cluster: c,
	})
	require.NoError(t, err)
	h.ratelimit = rl

	return &h
}

func (h *Harness) Register(register func(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context)))) {
	register(h.api, routes.Services{
		Logger:    h.logger,
		Metrics:   h.metrics,
		Vault:     nil,
		Ratelimit: h.ratelimit,
	})
}

func (h *Harness) Api() humatest.TestAPI {
	return h.api
}

// Post is a helper function to make a POST request to the API.
// It will hanndle serializing the request and response objects to and from JSON.
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body) {

	err := json.Unmarshal(r.Body.Bytes(), &body)
	require.NoError(t, err)

}
