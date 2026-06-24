package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		CtrlClient: &testutil.MockProjectClient{},
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.create_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("create basic project", func(t *testing.T) {
		req := handler.Request{Name: "Payments Service", Slug: "payments-service"}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.True(t, strings.HasPrefix(res.Body.Data.Id, "proj_"))
	})

	t.Run("create multiple projects with unique slugs", func(t *testing.T) {
		slugs := []string{uid.New("test"), uid.New("test"), uid.New("test")}
		for _, slug := range slugs {
			slug = strings.ToLower(strings.ReplaceAll(slug, "_", "-"))
			req := handler.Request{Name: slug, Slug: slug}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.True(t, strings.HasPrefix(res.Body.Data.Id, "proj_"))
		}
	})
}
