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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_update_namespace"
)

func TestUpdateNamespaceConflict(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyNamespace(workspace.ID)))

	deleted := seedNamespace(t, h, workspace.ID)
	require.NoError(t, db.Query.SoftDeleteRatelimitNamespace(t.Context(), h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{
		ID:  deleted.ID,
		Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))

	for name, taken := range map[string]string{
		"name in use":                 seedNamespace(t, h, workspace.ID).Name,
		"name of a deleted namespace": deleted.Name,
	} {
		t.Run(name, func(t *testing.T) {
			ns := seedNamespace(t, h, workspace.ID)

			res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{Namespace: ns.ID, Name: taken})
			require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
			require.Contains(t, res.Body.Error.Detail, taken)
			require.Equal(t, ns.Name, findNamespace(t, h, ns).Name)
		})
	}
}
