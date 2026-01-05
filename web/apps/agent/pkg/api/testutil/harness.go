package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/validation"
	"github.com/unkeyed/unkey/svc/agent/pkg/cluster"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/membership"
	"github.com/unkeyed/unkey/svc/agent/pkg/metrics"
	"github.com/unkeyed/unkey/svc/agent/pkg/port"
	"github.com/unkeyed/unkey/svc/agent/pkg/uid"
	"github.com/unkeyed/unkey/svc/agent/services/ratelimit"
)

type Harness struct {
	t *testing.T

	logger  logging.Logger
	metrics metrics.Metrics

	ratelimit ratelimit.Service

	mux *http.ServeMux
}

func NewHarness(t *testing.T) *Harness {
	mux := http.NewServeMux()

	p := port.New()
	nodeId := uid.New("test")
	authToken := uid.New("test")
	serfAddr := fmt.Sprintf("localhost:%d", p.Get())
	rpcAddr := fmt.Sprintf("localhost:%d", p.Get())

	h := Harness{
		t:       t,
		logger:  logging.NewNoopLogger(),
		metrics: metrics.NewNoop(),
		mux:     mux,
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

func (h *Harness) Register(route *routes.Route) {

	route.Register(h.mux)

}

func (h *Harness) SetupRoute(constructor func(svc routes.Services) *routes.Route) *routes.Route {

	validator, err := validation.New()
	require.NoError(h.t, err)
	route := constructor(routes.Services{
		Logger:           h.logger,
		Metrics:          h.metrics,
		Ratelimit:        h.ratelimit,
		Vault:            nil,
		OpenApiValidator: validator,
		Sender:           routes.NewJsonSender(h.logger),
	})
	h.Register(route)
	return route
}

// Post is a helper function to make a POST request to the API.
// It will hanndle serializing the request and response objects to and from JSON.
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body) {

	err := json.Unmarshal(r.Body.Bytes(), &body)
	require.NoError(t, err)

}

type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    TBody
}

func CallRoute[Req any, Res any](t *testing.T, route *routes.Route, headers http.Header, req Req) TestResponse[Res] {
	t.Helper()
	mux := http.NewServeMux()
	route.Register(mux)

	rr := httptest.NewRecorder()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	require.NoError(t, err)

	httpReq := httptest.NewRequest(route.Method(), route.Path(), body)
	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}
	if route.Method() == http.MethodPost {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	mux.ServeHTTP(rr, httpReq)
	require.NoError(t, err)

	var res Res
	err = json.NewDecoder(rr.Body).Decode(&res)
	require.NoError(t, err)

	return TestResponse[Res]{
		Status:  rr.Code,
		Headers: rr.Header(),
		Body:    res,
	}
}
