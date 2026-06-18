package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Auditlogs:  h.Auditlogs,
		CtrlClient: &testutil.MockProjectClient{},
	}

	h.Register(route)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "exact permission", permissions: []string{"project.*.create_project"}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "project.*.create_project"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"project.*.read_project"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.create_api"}, shouldPass: false},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			req := handler.Request{Name: tc.name, Slug: fmt.Sprintf("forbidden-test-%d", i)}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			}
		})
	}
}
