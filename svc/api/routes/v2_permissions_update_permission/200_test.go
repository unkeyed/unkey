package handler_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func TestUpdatePermissionSuccess(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyPermission(workspace.ID)))

	countUpdateLogs := func(t *testing.T, permissionID string) int {
		t.Helper()
		count := 0
		for _, ev := range h.FindAuditLogsByTargetID(ctx, t, permissionID) {
			if ev.Event == string(auditlog.PermissionUpdateEvent) {
				require.Equal(t, workspace.ID, ev.WorkspaceID)
				count++
			}
		}
		return count
	}

	t.Run("rename by id keeps slug and description", func(t *testing.T) {
		permission := seedPermission(h, workspace.ID)
		name := "Read documents"

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permission: permission.ID,
			Name:       &name,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, permission.ID, res.Body.Data.Id)
		require.Equal(t, name, res.Body.Data.Name)
		require.Equal(t, permission.Slug, res.Body.Data.Slug)
		require.Equal(t, "KEBAP", res.Body.Data.Description)

		stored := findPermission(t, h, permission)
		require.Equal(t, name, stored.Name)
		require.Equal(t, permission.Slug, stored.Slug)
		require.Equal(t, "KEBAP", stored.Description.String)
		require.True(t, stored.UpdatedAtM.Valid)
		require.Equal(t, 1, countUpdateLogs(t, permission.ID))

		for _, ev := range h.FindAuditLogsByTargetID(ctx, t, permission.ID) {
			if ev.Event != string(auditlog.PermissionUpdateEvent) {
				continue
			}
			require.Len(t, ev.Targets, 1)
			require.Equal(t, name, ev.Targets[0].Name)
			require.Equal(t, name, ev.Targets[0].Meta["name"])
			require.Equal(t, permission.Slug, ev.Targets[0].Meta["slug"])
			require.Equal(t, "KEBAP", ev.Targets[0].Meta["description"])
		}
	})

	t.Run("id match wins over a slug that equals the id", func(t *testing.T) {
		target := seedPermission(h, workspace.ID)
		decoy := seedPermission(h, workspace.ID)
		decoySlug := target.ID
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permission: decoy.ID,
			Slug:       &decoySlug,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		name := "Read documents"
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permission: target.ID,
			Name:       &name,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, target.ID, res.Body.Data.Id)
		require.Equal(t, name, findPermission(t, h, target).Name)
		require.Equal(t, decoy.Name, findPermission(t, h, decoy).Name)
	})

	t.Run("change slug located by slug", func(t *testing.T) {
		permission := seedPermission(h, workspace.ID)
		slug := randomSlug() + ".read"

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permission: permission.Slug,
			Slug:       &slug,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, slug, res.Body.Data.Slug)

		stored := findPermission(t, h, permission)
		require.Equal(t, slug, stored.Slug)
		require.Equal(t, permission.Name, stored.Name)
	})

	t.Run("update every field", func(t *testing.T) {
		permission := seedPermission(h, workspace.ID)
		name := "Write documents"
		slug := randomSlug()

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permission:  permission.ID,
			Name:        &name,
			Slug:        &slug,
			Description: nullable.NewNullableWithValue("Allows writing documents"),
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		stored := findPermission(t, h, permission)
		require.Equal(t, name, stored.Name)
		require.Equal(t, slug, stored.Slug)
		require.Equal(t, "Allows writing documents", stored.Description.String)
	})

	t.Run("null and empty description clear it", func(t *testing.T) {
		for _, description := range []nullable.Nullable[string]{
			nullable.NewNullNullable[string](),
			nullable.NewNullableWithValue(""),
		} {
			permission := seedPermission(h, workspace.ID)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Permission:  permission.ID,
				Description: description,
			})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Empty(t, res.Body.Data.Description)

			stored := findPermission(t, h, permission)
			require.False(t, stored.Description.Valid)
			require.Equal(t, permission.Name, stored.Name)
		}
	})

	t.Run("concurrent partial updates both persist", func(t *testing.T) {
		for range 10 {
			permission := seedPermission(h, workspace.ID)
			name := uid.New("name")
			description := uid.New("description")

			var wg sync.WaitGroup
			statuses := make([]int, 2)
			wg.Go(func() {
				statuses[0] = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
					Permission: permission.ID,
					Name:       &name,
				}).Status
			})
			wg.Go(func() {
				statuses[1] = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
					Permission:  permission.Slug,
					Description: nullable.NewNullableWithValue(description),
				}).Status
			})
			wg.Wait()

			require.Equal(t, []int{http.StatusOK, http.StatusOK}, statuses)
			stored := findPermission(t, h, permission)
			require.Equal(t, name, stored.Name)
			require.Equal(t, description, stored.Description.String)
		}
	})

	t.Run("no fields writes nothing", func(t *testing.T) {
		permission := seedPermission(h, workspace.ID)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permission: permission.ID,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, permission.Name, res.Body.Data.Name)
		require.Equal(t, permission.Slug, res.Body.Data.Slug)
		require.Equal(t, "KEBAP", res.Body.Data.Description)

		stored := findPermission(t, h, permission)
		require.False(t, stored.UpdatedAtM.Valid)
		require.Zero(t, countUpdateLogs(t, permission.ID))
	})

	t.Run("repeating a request succeeds", func(t *testing.T) {
		permission := seedPermission(h, workspace.ID)
		slug := randomSlug()
		req := handler.Request{Permission: permission.ID, Slug: &slug}

		for range 2 {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		}
		require.Equal(t, slug, findPermission(t, h, permission).Slug)
	})
}
