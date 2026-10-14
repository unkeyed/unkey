package handler_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_update_namespace"
)

func TestUpdateNamespaceNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyNamespace(workspace.ID)))

	call := func(t *testing.T, namespace string) {
		t.Helper()
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Namespace: namespace, Name: newName()})
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
		require.Equal(t, ns.Name, findNamespace(t, h, ns).Name)
	})

	t.Run("deleted namespace", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		require.NoError(t, db.Query.SoftDeleteRatelimitNamespace(t.Context(), h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{
			ID:  ns.ID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))
		call(t, ns.ID)
		call(t, ns.Name)
		require.Equal(t, ns.Name, findNamespace(t, h, ns).Name)
	})
}
