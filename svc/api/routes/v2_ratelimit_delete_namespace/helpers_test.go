package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_delete_namespace"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		DB:             h.DB,
		Auditlogs:      h.Auditlogs,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)
	return route
}

func seedNamespace(t *testing.T, h *testutil.Harness, workspaceID string) db.InsertRatelimitNamespaceParams {
	t.Helper()
	projectID, err := projects.EnsureDefaultProject(t.Context(), h.DB.RW(), workspaceID)
	require.NoError(t, err)
	ns := db.InsertRatelimitNamespaceParams{
		ID:          uid.New(uid.RatelimitNamespacePrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        uid.New("email.outbound"),
		CreatedAt:   time.Now().UnixMilli(),
	}
	require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), ns))
	return ns
}

func seedOverride(t *testing.T, h *testutil.Harness, ns db.InsertRatelimitNamespaceParams) string {
	t.Helper()
	id := uid.New(uid.RatelimitOverridePrefix)
	require.NoError(t, db.Query.InsertRatelimitOverride(t.Context(), h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          id,
		WorkspaceID: ns.WorkspaceID,
		NamespaceID: ns.ID,
		Identifier:  uid.New("KEBAP"),
		Limit:       10,
		Duration:    1000,
		CreatedAt:   time.Now().UnixMilli(),
	}))
	return id
}

func isDeleted(t *testing.T, h *testutil.Harness, ns db.InsertRatelimitNamespaceParams) bool {
	t.Helper()
	stored, err := db.Query.FindRatelimitNamespace(t.Context(), h.DB.RO(), db.FindRatelimitNamespaceParams{
		WorkspaceID: ns.WorkspaceID,
		Namespace:   ns.ID,
	})
	require.NoError(t, err)
	return stored.DeletedAtM.Valid
}

func deleteAnyNamespace(workspaceID string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/*/ratelimits/namespaces/*#delete", workspaceID)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
