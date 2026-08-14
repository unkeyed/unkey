package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test400_MalformedCursor(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "audit.*.read_audit_log")
	route := newRoute(h)
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{
		Cursor: ptr.P("!!!not-a-valid-cursor!!!"),
	})
	require.Equal(t, http.StatusBadRequest, res.Status)
}
