package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_list_namespaces"
)

func TestListNamespacesBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	headers := authHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project("*").RatelimitNamespace("*"))))

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "limit below minimum", req: handler.Request{Limit: new(0)}},
		{name: "limit above maximum", req: handler.Request{Limit: new(101)}},
		{name: "empty cursor", req: handler.Request{Cursor: new("")}},
		{name: "cursor too long", req: handler.Request{Cursor: new(strings.Repeat("a", 1025))}},
		{name: "search too long", req: handler.Request{Search: new(strings.Repeat("a", 257))}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
