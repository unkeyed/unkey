package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_usage"
)

func TestGetUsageUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers("invalid_token"), handler.Request{})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
