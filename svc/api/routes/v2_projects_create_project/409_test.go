package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectDuplicate(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.create_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("same slug twice returns 409", func(t *testing.T) {
		req := handler.Request{Name: "Payments Service", Slug: "duplicate-slug"}

		successRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, successRes.Status, "expected 200, received: %s", successRes.RawBody)

		errorRes := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, req)
		require.Equal(t, 409, errorRes.Status, "expected 409, received: %s", errorRes.RawBody)
		require.NotNil(t, errorRes.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_already_exists", errorRes.Body.Error.Type)
	})

	t.Run("same slug in different workspace succeeds", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		otherRootKey := h.CreateRootKey(otherWorkspace.ID, "project.*.create_project")
		otherHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", otherRootKey)},
		}

		req := handler.Request{Name: "Payments Service", Slug: "shared-slug"}

		first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, first.Status, "expected 200, received: %s", first.RawBody)

		second := testutil.CallRoute[handler.Request, handler.Response](h, route, otherHeaders, req)
		require.Equal(t, 200, second.Status, "expected 200 for other workspace, received: %s", second.RawBody)
		require.Equal(t, req.Slug, second.Body.Data.Slug)
	})
}
