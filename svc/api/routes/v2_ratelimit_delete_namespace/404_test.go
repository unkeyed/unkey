package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_delete_namespace"
)

func TestDeleteNamespaceNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, deleteAnyNamespace(workspace.ID)))

	call := func(t *testing.T, namespace string) {
		t.Helper()
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Namespace: namespace})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	}

	t.Run("missing namespace", func(t *testing.T) {
		call(t, uid.New(uid.RatelimitNamespacePrefix))
	})

	t.Run("namespace in another workspace", func(t *testing.T) {
		ns := seedNamespace(t, h, h.CreateWorkspace().ID)
		call(t, ns.ID)
		call(t, ns.Name)
		require.False(t, isDeleted(t, h, ns))
	})

	t.Run("already deleted namespace", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		call(t, ns.ID)
		call(t, ns.Name)
	})
}
