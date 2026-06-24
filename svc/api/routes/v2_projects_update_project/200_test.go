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
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_update_project"
)

func TestUpdateProjectSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.update_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	createProject := func(t *testing.T, name string, deleteProtection bool) (string, string) {
		t.Helper()
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      workspace.ID,
			Name:             name,
			Slug:             slug,
			DeleteProtection: deleteProtection,
		})
		return project.ID, slug
	}

	getProject := func(t *testing.T, id string) db.FindProjectByWorkspaceAndSlugRow {
		t.Helper()
		project, err := db.ResolveProject(ctx, h.DB.RO(), workspace.ID, id)
		require.NoError(t, err)
		return project
	}

	t.Run("update name only", func(t *testing.T) {
		id, slug := createProject(t, "Old Name", false)
		newName := "New Name"

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: id,
			Name:    &newName,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, id, res.Body.Data.Id)
		require.Equal(t, newName, res.Body.Data.Name)
		require.Equal(t, slug, res.Body.Data.Slug)
		require.False(t, res.Body.Data.DeleteProtection)
		require.Greater(t, res.Body.Data.UpdatedAt, int64(0))

		project := getProject(t, id)
		require.Equal(t, newName, project.Name)
		require.Equal(t, slug, project.Slug)
		require.False(t, project.DeleteProtection.Bool)
		require.True(t, project.UpdatedAt.Valid)
		require.Greater(t, project.UpdatedAt.Int64, int64(0))

		auditLogs := h.FindAuditLogsByTargetID(ctx, t, id)
		var found bool
		for _, ev := range auditLogs {
			if ev.Event == "project.update" {
				found = true
				require.Equal(t, workspace.ID, ev.WorkspaceID)
				break
			}
		}
		require.True(t, found, "should find a project.update audit log event")
	})

	t.Run("update slug only", func(t *testing.T) {
		id, _ := createProject(t, "Slug Change", false)
		newSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: id,
			Slug:    &newSlug,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newSlug, res.Body.Data.Slug)
		require.Equal(t, "Slug Change", res.Body.Data.Name, "name must survive a slug-only update")

		project := getProject(t, id)
		require.Equal(t, newSlug, project.Slug)
		require.Equal(t, "Slug Change", project.Name)
	})

	t.Run("locate by slug and change slug", func(t *testing.T) {
		id, slug := createProject(t, "Located By Slug", false)
		newSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: slug,
			Slug:    &newSlug,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, id, res.Body.Data.Id, "must resolve to the same project via its slug")
		require.Equal(t, newSlug, res.Body.Data.Slug)

		project := getProject(t, id)
		require.Equal(t, newSlug, project.Slug)
	})

	t.Run("update delete protection only", func(t *testing.T) {
		id, _ := createProject(t, "Keep Name", false)
		protect := true

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:          id,
			DeleteProtection: &protect,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, "Keep Name", res.Body.Data.Name)
		require.True(t, res.Body.Data.DeleteProtection)

		project := getProject(t, id)
		require.Equal(t, "Keep Name", project.Name)
		require.True(t, project.DeleteProtection.Bool)
	})

	t.Run("omitted fields keep non-default values", func(t *testing.T) {
		id, _ := createProject(t, "Original", true)

		newName := "Renamed"
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: id,
			Name:    &newName,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newName, res.Body.Data.Name)
		require.True(t, res.Body.Data.DeleteProtection, "delete protection must survive a name-only update")

		project := getProject(t, id)
		require.Equal(t, newName, project.Name)
		require.True(t, project.DeleteProtection.Bool)

		unprotect := false
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:          id,
			DeleteProtection: &unprotect,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newName, res.Body.Data.Name, "name must survive a delete-protection-only update")
		require.False(t, res.Body.Data.DeleteProtection)

		project = getProject(t, id)
		require.Equal(t, newName, project.Name)
		require.False(t, project.DeleteProtection.Bool)
	})

	t.Run("update both name and delete protection", func(t *testing.T) {
		id, _ := createProject(t, "Before", true)
		newName := "After"
		unprotect := false

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:          id,
			Name:             &newName,
			DeleteProtection: &unprotect,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newName, res.Body.Data.Name)
		require.False(t, res.Body.Data.DeleteProtection)

		project := getProject(t, id)
		require.Equal(t, newName, project.Name)
		require.False(t, project.DeleteProtection.Bool)
	})

	t.Run("empty body leaves project unchanged", func(t *testing.T) {
		id, slug := createProject(t, "Unchanged", true)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: id,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, "Unchanged", res.Body.Data.Name)
		require.Equal(t, slug, res.Body.Data.Slug)
		require.True(t, res.Body.Data.DeleteProtection)

		project := getProject(t, id)
		require.Equal(t, "Unchanged", project.Name)
		require.Equal(t, slug, project.Slug)
		require.True(t, project.DeleteProtection.Bool)
	})
}
