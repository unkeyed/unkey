package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_update_namespace"
)

func TestUpdateNamespaceUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	ns := seedNamespace(t, h, h.Resources().UserWorkspace.ID)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, authHeaders("invalid_token"), handler.Request{Namespace: ns.ID, Name: newName()})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
	require.Equal(t, ns.Name, findNamespace(t, h, ns).Name)
}
