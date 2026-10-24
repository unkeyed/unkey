package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_exchange_token"
)

func TestExchangeRequiresExactlyOneCredentialWithoutMiddleware(t *testing.T) {
	cases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing", req: handler.Request{}},
		{name: "mixed", req: handler.Request{LoginId: new(uid.New(uid.DeviceAuthorizationPrefix)), RefreshToken: new("invalid-refresh-token")}},
		{name: "empty mixed", req: handler.Request{LoginId: new(""), RefreshToken: new("")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.req)
			require.NoError(t, err)
			route := &handler.Handler{}
			req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), req, 1<<20))
			err = route.Handle(t.Context(), session)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.App.Validation.InvalidInput.URN(), code)
		})
	}
}

func TestExchangeRejectsInvalidCredentialsOverHTTP(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)
	cases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing", req: handler.Request{}},
		{name: "mixed", req: handler.Request{LoginId: new(uid.New(uid.DeviceAuthorizationPrefix)), RefreshToken: new("invalid-refresh-token")}},
		{name: "empty login", req: handler.Request{LoginId: new("")}},
		{name: "empty refresh", req: handler.Request{RefreshToken: new("")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, jsonHeaders, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
		})
	}
}
