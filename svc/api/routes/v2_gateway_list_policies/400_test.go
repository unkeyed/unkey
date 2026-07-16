package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_list_policies"
)

func TestListPoliciesBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_policies")
	headers := authHeaders(rootKey)

	callTyped := func(t *testing.T, req handler.Request) testutil.TestResponse[openapi.BadRequestErrorResponse] {
		t.Helper()
		return testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	}

	t.Run("missing identifiers", func(t *testing.T) {
		res := callTyped(t, handler.Request{
			Project:     "",
			App:         "",
			Environment: "",
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})
}
