package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_create_namespace"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)
	return route
}

func newName() string {
	return uid.New("email.outbound")
}

func findNamespace(t *testing.T, h *testutil.Harness, workspaceID, name string) (db.RatelimitNamespace, bool) {
	t.Helper()
	ns, err := db.Query.FindRatelimitNamespaceByName(t.Context(), h.DB.RO(), db.FindRatelimitNamespaceByNameParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if db.IsNotFound(err) {
		return db.RatelimitNamespace{}, false
	}
	require.NoError(t, err)
	return ns, true
}

func writeAnyNamespace(workspaceID string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/*/ratelimits/namespaces/*#write", workspaceID)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
