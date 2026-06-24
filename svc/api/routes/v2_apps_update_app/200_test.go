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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
)

func TestUpdateAppSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.update_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})

	createApp := func(t *testing.T, name, defaultBranch string) (string, string) {
		t.Helper()
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		app := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          name,
			Slug:          slug,
			DefaultBranch: defaultBranch,
		})
		return app.ID, slug
	}

	getApp := func(t *testing.T, id string) db.App {
		t.Helper()
		app, err := db.Query.FindAppById(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		return app
	}

	t.Run("update name only", func(t *testing.T) {
		id, slug := createApp(t, "Old Name", "main")
		newName := "New Name"

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Name:    &newName,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, id, res.Body.Data.Id)
		require.Equal(t, newName, res.Body.Data.Name)
		require.Equal(t, slug, res.Body.Data.Slug)
		require.Equal(t, "main", res.Body.Data.DefaultBranch)
		require.Equal(t, project.ID, res.Body.Data.ProjectId)
		require.False(t, res.Body.Data.DeleteProtection)
		require.Greater(t, res.Body.Data.UpdatedAt, int64(0))

		app := getApp(t, id)
		require.Equal(t, newName, app.Name)
		require.Equal(t, slug, app.Slug)
		require.Equal(t, "main", app.DefaultBranch)
		require.True(t, app.UpdatedAt.Valid)
		require.Greater(t, app.UpdatedAt.Int64, int64(0))

		auditLogs := h.FindAuditLogsByTargetID(ctx, t, id)
		var found bool
		for _, ev := range auditLogs {
			if ev.Event == "app.update" {
				found = true
				require.Equal(t, workspace.ID, ev.WorkspaceID)
				break
			}
		}
		require.True(t, found, "should find an app.update audit log event")
	})

	t.Run("update slug only", func(t *testing.T) {
		id, _ := createApp(t, "Slug Change", "main")
		newSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Slug:    &newSlug,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newSlug, res.Body.Data.Slug)
		require.Equal(t, "Slug Change", res.Body.Data.Name, "name must survive a slug-only update")

		app := getApp(t, id)
		require.Equal(t, newSlug, app.Slug)
		require.Equal(t, "Slug Change", app.Name)
	})

	t.Run("update default branch only", func(t *testing.T) {
		id, _ := createApp(t, "Branch Change", "main")

		newBranch := "develop"
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:       project.ID,
			App:           id,
			DefaultBranch: &newBranch,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, "develop", res.Body.Data.DefaultBranch)
		require.Equal(t, "Branch Change", res.Body.Data.Name, "name must survive a branch-only update")

		app := getApp(t, id)
		require.Equal(t, "develop", app.DefaultBranch)
		require.Equal(t, "Branch Change", app.Name)
	})

	t.Run("update delete protection only", func(t *testing.T) {
		id, _ := createApp(t, "Keep Name", "main")
		protect := true

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:          project.ID,
			App:              id,
			DeleteProtection: &protect,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, "Keep Name", res.Body.Data.Name)
		require.True(t, res.Body.Data.DeleteProtection)

		app := getApp(t, id)
		require.Equal(t, "Keep Name", app.Name)
		require.True(t, app.DeleteProtection.Bool)
	})

	t.Run("omitted fields keep non-default values", func(t *testing.T) {
		id, _ := createApp(t, "Original", "main")

		// Seeded apps start with delete protection off; turn it on so the
		// subsequent name-only update has a non-default value to preserve.
		protect := true
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:          project.ID,
			App:              id,
			DeleteProtection: &protect,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.DeleteProtection)

		newName := "Renamed"
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Name:    &newName,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newName, res.Body.Data.Name)
		require.True(t, res.Body.Data.DeleteProtection, "delete protection must survive a name-only update")

		app := getApp(t, id)
		require.Equal(t, newName, app.Name)
		require.True(t, app.DeleteProtection.Bool)
	})

	t.Run("update all fields together", func(t *testing.T) {
		id, _ := createApp(t, "Before", "main")
		newName := "After"
		newSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		newBranch := "release"
		protect := true

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:          project.ID,
			App:              id,
			Name:             &newName,
			Slug:             &newSlug,
			DefaultBranch:    &newBranch,
			DeleteProtection: &protect,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, newName, res.Body.Data.Name)
		require.Equal(t, newSlug, res.Body.Data.Slug)
		require.Equal(t, newBranch, res.Body.Data.DefaultBranch)
		require.True(t, res.Body.Data.DeleteProtection)

		app := getApp(t, id)
		require.Equal(t, newName, app.Name)
		require.Equal(t, newSlug, app.Slug)
		require.Equal(t, newBranch, app.DefaultBranch)
		require.True(t, app.DeleteProtection.Bool)
	})

	t.Run("empty body leaves app unchanged", func(t *testing.T) {
		id, slug := createApp(t, "Unchanged", "main")

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, "Unchanged", res.Body.Data.Name)
		require.Equal(t, slug, res.Body.Data.Slug)
		require.Equal(t, "main", res.Body.Data.DefaultBranch)
		require.False(t, res.Body.Data.DeleteProtection)

		app := getApp(t, id)
		require.Equal(t, "Unchanged", app.Name)
		require.Equal(t, slug, app.Slug)
		require.Equal(t, "main", app.DefaultBranch)
	})
}
