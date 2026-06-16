package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectSuccessfully(t *testing.T) {
	ctx := context.Background()
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

	t.Run("create basic project", func(t *testing.T) {
		req := handler.Request{Name: "Payments Service", Slug: "payments-service"}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, req.Slug, res.Body.Data.Slug)
		require.True(t, strings.HasPrefix(res.Body.Data.Id, "proj_"))

		project, err := db.Query.FindProjectByWorkspaceAndSlug(ctx, h.DB.RO(), db.FindProjectByWorkspaceAndSlugParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Slug:        req.Slug,
		})
		require.NoError(t, err)
		require.Equal(t, req.Name, project.Name)
		require.Equal(t, h.Resources().UserWorkspace.ID, project.WorkspaceID)
		require.True(t, strings.HasPrefix(project.ID, "proj_"))
		require.Equal(t, project.ID, res.Body.Data.Id)

		auditLogs := h.FindAuditLogsByTargetID(ctx, t, project.ID)
		var foundCreateEvent bool
		for _, ev := range auditLogs {
			if ev.Event == "project.create" {
				foundCreateEvent = true
				require.Equal(t, h.Resources().UserWorkspace.ID, ev.WorkspaceID)
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find a project.create audit log event")
	})

	t.Run("create multiple projects with unique slugs", func(t *testing.T) {
		slugs := []string{uid.New("test"), uid.New("test"), uid.New("test")}
		for _, slug := range slugs {
			slug = strings.ToLower(strings.ReplaceAll(slug, "_", "-"))
			req := handler.Request{Name: slug, Slug: slug}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, slug, res.Body.Data.Slug)
		}
	})
}
