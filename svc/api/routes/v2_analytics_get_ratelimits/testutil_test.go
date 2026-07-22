package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func newRoute(t *testing.T, analytics bool) (*testutil.Harness, *Handler, string) {
	t.Helper()

	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	if analytics {
		h.SetupAnalytics(workspace.ID)
	}
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)
	return h, route, workspace.ID
}

func createNamespace(t *testing.T, h *testutil.Harness, workspaceID string) string {
	t.Helper()

	id := uid.New(uid.RatelimitNamespacePrefix)
	require.NoError(t, db.Query.InsertRatelimitNamespace(context.Background(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        uid.New("test"),
		CreatedAt:   time.Now().UnixMilli(),
	}))
	return id
}

func auth(rootKey string) http.Header {
	return http.Header{"Authorization": {fmt.Sprintf("Bearer %s", rootKey)}, "Content-Type": {"application/json"}}
}
