package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_exchange_token"
)

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
		{name: "empty mixed", req: handler.Request{LoginId: new(""), RefreshToken: new("")}},
		{name: "empty login", req: handler.Request{LoginId: new("")}},
		{name: "empty refresh", req: handler.Request{RefreshToken: new("")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, jsonHeaders, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
			require.Equal(t, codes.App.Validation.InvalidInput.DocsURL(), res.Body.Error.Type)
		})
	}
}
