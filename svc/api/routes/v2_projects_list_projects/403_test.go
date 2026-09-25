package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_list_projects"
)

// TestListProjectsFiltersUnauthorizedRows guarantees unrelated permissions do
// not reject a list request. For example, create_project alone returns no
// projects because it does not permit reading any row.
func TestListProjectsFiltersUnauthorizedRows(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "exact permission", permissions: []string{"project.*.read_project"}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "project.*.read_project"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"project.*.create_project"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
			require.Equal(t, http.StatusOK, res.Status, "permissions: %v, response: %s", tc.permissions, res.RawBody)
			if !tc.shouldPass {
				require.Empty(t, res.Body.Data)
				require.Nil(t, res.Body.Pagination.Cursor)
				require.False(t, res.Body.Pagination.HasMore)
			}
		})
	}
}
