package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environments"
)

func TestListEnvironmentsValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_environment")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing project", req: handler.Request{App: "payments-api"}},
		{name: "missing app", req: handler.Request{Project: "payments-service"}},
		{name: "project invalid chars", req: handler.Request{Project: "payments.service", App: "payments-api"}},
		{name: "app invalid chars", req: handler.Request{Project: "payments-service", App: "payments.api"}},
		{name: "app too long", req: handler.Request{Project: "payments-service", App: strings.Repeat("a", 256)}},
		{name: "project too long", req: handler.Request{Project: strings.Repeat("a", 257), App: "payments-api"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		})
	}
}
