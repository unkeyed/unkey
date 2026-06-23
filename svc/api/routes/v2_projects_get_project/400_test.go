//nolint:exhaustruct
package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_get_project"
)

func TestGetProjectBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing projectId", req: handler.Request{}},
		{name: "projectId too short", req: handler.Request{Project: "proj_1"}},
		{name: "projectId with invalid chars", req: handler.Request{Project: "proj-1234abc"}},
		{name: "projectId too long", req: handler.Request{Project: strings.Repeat("a", 256)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, "Bad Request", res.Body.Error.Title)
			require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
			require.Greater(t, len(res.Body.Error.Errors), 0)
		})
	}

	t.Run("invalid json", func(t *testing.T) {
		invalidJSON := `{"projectId": }`

		req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(invalidJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
	})
}
