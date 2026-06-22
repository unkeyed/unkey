package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_list_apps"
)

func TestListAppsValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	validProject := "payments-service"

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing project", req: handler.Request{Limit: ptr.P(10)}},
		{name: "project invalid chars", req: handler.Request{Project: "Payments_Service"}},
		{name: "project too long", req: handler.Request{Project: strings.Repeat("a", 257)}},
		{name: "limit below minimum", req: handler.Request{Project: validProject, Limit: ptr.P(0)}},
		{name: "limit above maximum", req: handler.Request{Project: validProject, Limit: ptr.P(101)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		})
	}
}
