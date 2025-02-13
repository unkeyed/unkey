package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

type Harness struct {
	t *testing.T

	srv        *zen.Server
	containers *Containers
	validator  *validation.Validator

	middleware []zen.Middleware

	DB     database.Database
	Logger logging.Logger
	Keys   keys.KeyService
}

func NewHarness(t *testing.T) *Harness {

	logger := logging.New(logging.Config{Development: true, NoColor: false})

	containers := NewContainers(t)

	dsn := containers.RunMySQL()

	db, err := database.New(database.Config{
		Logger:     logger,
		PrimaryDSN: dsn,
	})
	require.NoError(t, err)

	srv, err := zen.New(zen.Config{
		NodeID: "test",
		Logger: logger,
	})
	require.NoError(t, err)

	keyService, err := keys.New(keys.Config{
		Logger: logger,
		DB:     db,
	})
	require.NoError(t, err)

	validator, err := validation.New()
	require.NoError(t, err)

	h := Harness{
		t:          t,
		Logger:     logger,
		srv:        srv,
		containers: containers,
		validator:  validator,
		Keys:       keyService,
		DB:         db,

		middleware: []zen.Middleware{
			zen.WithTracing(),
			//	zen.WithMetrics(svc.EventBuffer)
			//zen.WithRootKeyAuth(keyService),
			zen.WithLogging(logger),
			zen.WithErrorHandling(),
			zen.WithValidation(validator),
		},
	}

	return &h
}

func (h *Harness) Register(route zen.Route, middleware ...zen.Middleware) {

	if len(middleware) == 0 {
		middleware = h.middleware
	}

	h.srv.RegisterRoute(
		middleware,
		route,
	)

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
