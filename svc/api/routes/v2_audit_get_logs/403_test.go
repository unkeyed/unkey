package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test403_NoAuditPermission(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	// A valid root key that lacks audit.*.read_audit_log.
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")
	route := newRoute(h)
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{})
	require.Equal(t, http.StatusForbidden, res.Status)
}
