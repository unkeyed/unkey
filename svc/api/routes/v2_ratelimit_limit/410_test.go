package v2RatelimitLimit_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_limit"
)

func TestSoftDeletedNamespace(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Ratelimit:               h.Ratelimit,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:               h.Auditlogs,
	}

	h.Register(route)

	// Test with soft-deleted namespace - should return 410
	t.Run("soft-deleted namespace returns 410", func(t *testing.T) {
		// Create a namespace and then soft-delete it
		deletedNamespace := uid.New("test")
		ctx := context.Background()

		// Create namespace
		namespaceID := uid.New(uid.RatelimitNamespacePrefix)
		err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
			ID:          namespaceID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        deletedNamespace,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Soft delete the namespace
		err = db.Query.SoftDeleteRatelimitNamespace(ctx, h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{
			ID:  namespaceID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		require.NoError(t, err)

		// Create root key with permissions to use the namespace
		rootKeyWithPerms := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headersWithPerms := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithPerms)},
		}

		req := handler.Request{
			Namespace:  deletedNamespace,
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
		}

		// Should return 410 because namespace is soft-deleted
		res := testutil.CallRoute[handler.Request, openapi.GoneErrorResponse](h, route, headersWithPerms, req)
		require.Equal(t, http.StatusGone, res.Status, "Should return 410 for soft-deleted namespace")
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_gone", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "deleted", "Error message should indicate namespace was deleted")
	})
}
