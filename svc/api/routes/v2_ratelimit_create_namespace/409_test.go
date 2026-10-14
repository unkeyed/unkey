package handler_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_create_namespace"
)

func TestCreateNamespaceConflict(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyNamespace(workspace.ID)))

	create := func(t *testing.T, name string) *handler.Response {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		return res.Body
	}

	t.Run("name in use", func(t *testing.T) {
		existing := create(t, newName())

		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{Name: existing.Data.Name})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Detail, existing.Data.Name)

		stored, found := findNamespace(t, h, workspace.ID, existing.Data.Name)
		require.True(t, found)
		require.Equal(t, existing.Data.Id, stored.ID)
	})

	t.Run("name of a deleted namespace", func(t *testing.T) {
		deleted := create(t, newName())
		require.NoError(t, db.Query.SoftDeleteRatelimitNamespace(t.Context(), h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{
			ID:  deleted.Data.Id,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{Name: deleted.Data.Name})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Detail, deleted.Data.Name)
	})
}
