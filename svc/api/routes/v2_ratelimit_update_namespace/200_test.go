package handler_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_update_namespace"
)

func TestUpdateNamespaceSuccess(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyNamespace(workspace.ID)))

	countUpdateLogs := func(t *testing.T, namespaceID string) int {
		t.Helper()
		count := 0
		for _, ev := range h.FindAuditLogsByTargetID(ctx, t, namespaceID) {
			if ev.Event == string(auditlog.RatelimitNamespaceUpdateEvent) {
				count++
			}
		}
		return count
	}

	t.Run("rename by id", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		name := newName()

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID, Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, ns.ID, res.Body.Data.Id)
		require.Equal(t, name, res.Body.Data.Name)
		require.Equal(t, ns.CreatedAtM, res.Body.Data.CreatedAt)
		require.NotZero(t, res.Body.Data.UpdatedAt)

		stored := findNamespace(t, h, ns)
		require.Equal(t, name, stored.Name)
		require.Equal(t, res.Body.Data.UpdatedAt, stored.UpdatedAtM.Int64)

		logs := h.FindAuditLogsByTargetID(ctx, t, ns.ID)
		require.Len(t, logs, 1)
		require.Equal(t, string(auditlog.RatelimitNamespaceUpdateEvent), logs[0].Event)
		require.Equal(t, workspace.ID, logs[0].WorkspaceID)
		require.Equal(t, name, logs[0].Targets[0].Name)
		require.Equal(t, ns.Name, logs[0].Targets[0].Meta["previousName"])
	})

	t.Run("drops cached entries for the id, old name and new name", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		name := newName()
		cached := db.FindRatelimitNamespace{ID: ns.ID, Name: ns.Name}
		idKey := cache.ScopedKey{WorkspaceID: workspace.ID, Key: ns.ID}
		oldNameKey := cache.ScopedKey{WorkspaceID: workspace.ID, Key: ns.Name}
		newNameKey := cache.ScopedKey{WorkspaceID: workspace.ID, Key: name}
		h.Caches.RatelimitNamespace.Set(ctx, idKey, cached)
		h.Caches.RatelimitNamespace.Set(ctx, oldNameKey, cached)
		h.Caches.RatelimitNamespace.SetNull(ctx, newNameKey)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID, Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		for _, key := range []cache.ScopedKey{idKey, oldNameKey, newNameKey} {
			_, hit := h.Caches.RatelimitNamespace.Get(ctx, key)
			require.Equal(t, cache.Miss, hit, key.Key)
		}
	})

	t.Run("rename by current name", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		name := newName()

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.Name, Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, name, findNamespace(t, h, ns).Name)
	})

	t.Run("id match wins over a name that equals the id", func(t *testing.T) {
		target := seedNamespace(t, h, workspace.ID)
		decoy := seedNamespace(t, h, workspace.ID)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: decoy.ID, Name: target.ID})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		name := newName()
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: target.ID, Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, target.ID, res.Body.Data.Id)
		require.Equal(t, name, findNamespace(t, h, target).Name)
		require.Equal(t, target.ID, findNamespace(t, h, decoy).Name)
	})

	t.Run("change name casing", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		name := strings.ToUpper(ns.Name)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID, Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, name, findNamespace(t, h, ns).Name)
	})

	t.Run("name at the 512 character column limit", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		name := strings.Repeat("ü", 512)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID, Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, name, findNamespace(t, h, ns).Name)
	})

	t.Run("same name writes nothing", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		idKey := cache.ScopedKey{WorkspaceID: workspace.ID, Key: ns.ID}
		h.Caches.RatelimitNamespace.Set(ctx, idKey, db.FindRatelimitNamespace{ID: ns.ID, Name: ns.Name})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID, Name: ns.Name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, ns.Name, res.Body.Data.Name)
		require.Zero(t, res.Body.Data.UpdatedAt)

		require.False(t, findNamespace(t, h, ns).UpdatedAtM.Valid)
		require.Zero(t, countUpdateLogs(t, ns.ID))
		_, hit := h.Caches.RatelimitNamespace.Get(ctx, idKey)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("repeating a request succeeds and writes once", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		req := handler.Request{Namespace: ns.ID, Name: newName()}

		for range 2 {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		}
		require.Equal(t, req.Name, findNamespace(t, h, ns).Name)
		require.Equal(t, 1, countUpdateLogs(t, ns.ID))
	})

	t.Run("concurrent renames of one namespace both succeed", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		names := []string{newName(), newName()}

		var wg sync.WaitGroup
		statuses := make([]int, 2)
		for i, name := range names {
			wg.Go(func() {
				statuses[i] = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID, Name: name}).Status
			})
		}
		wg.Wait()

		require.Equal(t, []int{http.StatusOK, http.StatusOK}, statuses)
		require.Contains(t, names, findNamespace(t, h, ns).Name)
		require.Equal(t, 2, countUpdateLogs(t, ns.ID))
	})
}
