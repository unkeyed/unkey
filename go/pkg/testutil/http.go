package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Harness struct {
	t *testing.T

	logger logging.Logger

	srv *zen.Server
}

func NewHarness(t *testing.T) *Harness {

	logger := logging.NewNoop()

	srv, err := zen.New(zen.Config{
		NodeID: "test",
		Logger: logger,
	})
	require.NoError(t, err)

	h := Harness{
		t:      t,
		logger: logger,
		srv:    srv,
	}

	return &h
}

func (h *Harness) Register(route zen.Route) {

	h.srv.RegisterRoute([]zen.Middleware{}, route)

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

func CallRoute[Req any, Res any](h *Harness, route zen.Route, headers http.Header, req Req) TestResponse[Res] {
	h.t.Helper()

	rr := httptest.NewRecorder()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	require.NoError(h.t, err)

	httpReq := httptest.NewRequest(route.Method(), route.Path(), body)
	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}
	if route.Method() == http.MethodPost {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	h.srv.Mux().ServeHTTP(rr, httpReq)
	require.NoError(h.t, err)

	var res Res
	err = json.NewDecoder(rr.Body).Decode(&res)
	require.NoError(h.t, err)

	return TestResponse[Res]{
		Status:  rr.Code,
		Headers: rr.Header(),
		Body:    res,
	}
}
