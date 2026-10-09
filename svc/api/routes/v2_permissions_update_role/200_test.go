package handler_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_role"
)

func TestUpdateRoleSuccess(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyRole(workspace.ID)))

	countUpdateLogs := func(t *testing.T, roleID string) int {
		t.Helper()
		count := 0
		for _, ev := range h.FindAuditLogsByTargetID(ctx, t, roleID) {
			if ev.Event == string(auditlog.RoleUpdateEvent) {
				require.Equal(t, workspace.ID, ev.WorkspaceID)
				count++
			}
		}
		return count
	}

	t.Run("rename by id keeps description", func(t *testing.T) {
		role := seedRole(h, workspace.ID)
		name := uid.New("support.readonly")

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Role: role.ID,
			Name: &name,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, role.ID, res.Body.Data.Id)
		require.Equal(t, name, res.Body.Data.Name)
		require.Equal(t, "KEBAP", res.Body.Data.Description)

		stored := findRole(t, h, role)
		require.Equal(t, name, stored.Name)
		require.Equal(t, "KEBAP", stored.Description.String)
		require.True(t, stored.UpdatedAtM.Valid)
		require.Equal(t, 1, countUpdateLogs(t, role.ID))

		for _, ev := range h.FindAuditLogsByTargetID(ctx, t, role.ID) {
			if ev.Event != string(auditlog.RoleUpdateEvent) {
				continue
			}
			require.Len(t, ev.Targets, 1)
			require.Equal(t, name, ev.Targets[0].Name)
			require.Equal(t, name, ev.Targets[0].Meta["name"])
			require.Equal(t, "KEBAP", ev.Targets[0].Meta["description"])
		}
	})

	t.Run("change description located by name", func(t *testing.T) {
		role := seedRole(h, workspace.ID)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Role:        role.Name,
			Description: nullable.NewNullableWithValue("Read-only access for support"),
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, role.Name, res.Body.Data.Name)

		stored := findRole(t, h, role)
		require.Equal(t, role.Name, stored.Name)
		require.Equal(t, "Read-only access for support", stored.Description.String)
	})

	t.Run("id match wins over a name that equals the id", func(t *testing.T) {
		target := seedRole(h, workspace.ID)
		decoy := seedRole(h, workspace.ID)
		decoyName := target.ID
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Role: decoy.ID,
			Name: &decoyName,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		name := uid.New("support.readonly")
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Role: target.ID,
			Name: &name,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, target.ID, res.Body.Data.Id)
		require.Equal(t, name, findRole(t, h, target).Name)
		require.Equal(t, target.ID, findRole(t, h, decoy).Name)
	})

	t.Run("change name casing of the same role", func(t *testing.T) {
		role := seedRole(h, workspace.ID)
		name := strings.ToUpper(role.Name)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Role: role.ID,
			Name: &name,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, name, findRole(t, h, role).Name)
	})

	t.Run("null and empty description clear it", func(t *testing.T) {
		for _, description := range []nullable.Nullable[string]{
			nullable.NewNullNullable[string](),
			nullable.NewNullableWithValue(""),
		} {
			role := seedRole(h, workspace.ID)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Role:        role.ID,
				Description: description,
			})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Empty(t, res.Body.Data.Description)

			stored := findRole(t, h, role)
			require.False(t, stored.Description.Valid)
			require.Equal(t, role.Name, stored.Name)
		}
	})

	t.Run("concurrent partial updates both persist", func(t *testing.T) {
		for range 10 {
			role := seedRole(h, workspace.ID)
			name := uid.New("name")
			description := uid.New("description")

			var wg sync.WaitGroup
			statuses := make([]int, 2)
			wg.Go(func() {
				statuses[0] = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
					Role: role.ID,
					Name: &name,
				}).Status
			})
			wg.Go(func() {
				statuses[1] = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
					Role:        role.ID,
					Description: nullable.NewNullableWithValue(description),
				}).Status
			})
			wg.Wait()

			require.Equal(t, []int{http.StatusOK, http.StatusOK}, statuses)
			stored := findRole(t, h, role)
			require.Equal(t, name, stored.Name)
			require.Equal(t, description, stored.Description.String)
		}
	})

	t.Run("no fields writes nothing", func(t *testing.T) {
		role := seedRole(h, workspace.ID)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Role: role.ID,
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, role.Name, res.Body.Data.Name)
		require.Equal(t, "KEBAP", res.Body.Data.Description)

		stored := findRole(t, h, role)
		require.False(t, stored.UpdatedAtM.Valid)
		require.Zero(t, countUpdateLogs(t, role.ID))
	})

	t.Run("repeating a request succeeds", func(t *testing.T) {
		role := seedRole(h, workspace.ID)
		name := uid.New("support.readonly")
		req := handler.Request{Role: role.ID, Name: &name}

		for range 2 {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		}
		require.Equal(t, name, findRole(t, h, role).Name)
	})
}
