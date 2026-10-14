package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_set_override"
)

func TestNamespaceNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:             h.DB,
		Auditlogs:      h.Auditlogs,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with non-existent namespace ID
	t.Run("namespace id not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  uid.New(uid.RatelimitNamespacePrefix),
			Identifier: "some_identifier",
			Limit:      10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	// Test with non-existent namespace name
	t.Run("namespace name not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  "nonexistent_namespace",
			Identifier: "some_identifier",
			Limit:      10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	t.Run("deleted namespace", func(t *testing.T) {
		workspaceID := h.Resources().UserWorkspace.ID
		namespaceID := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
			ID:          namespaceID,
			WorkspaceID: workspaceID,
			Name:        uid.New("test"),
			CreatedAt:   time.Now().UnixMilli(),
		}))
		overrideID := uid.New(uid.RatelimitOverridePrefix)
		identifier := uid.New("KEBAP")
		require.NoError(t, db.Query.InsertRatelimitOverride(t.Context(), h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: workspaceID,
			NamespaceID: namespaceID,
			Identifier:  identifier,
			Limit:       10,
			Duration:    1000,
			CreatedAt:   time.Now().UnixMilli(),
		}))
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		require.NoError(t, db.Query.SoftDeleteRatelimitNamespace(t.Context(), h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{ID: namespaceID, Now: now}))
		require.NoError(t, db.Query.SoftDeleteRatelimitOverride(t.Context(), h.DB.RW(), db.SoftDeleteRatelimitOverrideParams{ID: overrideID, Now: now}))

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
			Namespace:  namespaceID,
			Identifier: identifier,
			Limit:      10,
			Duration:   1000,
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)

		override, err := db.Query.FindRatelimitOverrideByID(t.Context(), h.DB.RO(), db.FindRatelimitOverrideByIDParams{WorkspaceID: workspaceID, OverrideID: overrideID})
		require.NoError(t, err)
		require.True(t, override.DeletedAtM.Valid)
	})
}
