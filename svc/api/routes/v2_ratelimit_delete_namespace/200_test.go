package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_delete_namespace"
)

func TestDeleteNamespaceSuccess(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, deleteAnyNamespace(workspace.ID)))

	t.Run("deletes the namespace and its overrides", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		overrideIDs := []string{seedOverride(t, h, ns), seedOverride(t, h, ns)}
		other := seedNamespace(t, h, workspace.ID)
		otherOverride := seedOverride(t, h, other)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, isDeleted(t, h, ns))
		require.False(t, isDeleted(t, h, other))

		for _, id := range overrideIDs {
			override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{WorkspaceID: workspace.ID, OverrideID: id})
			require.NoError(t, err)
			require.True(t, override.DeletedAtM.Valid)
		}
		override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{WorkspaceID: workspace.ID, OverrideID: otherOverride})
		require.NoError(t, err)
		require.False(t, override.DeletedAtM.Valid)

		namespaceLogs := h.FindAuditLogsByTargetID(ctx, t, ns.ID)
		require.Len(t, namespaceLogs, 3)
		correlationID := namespaceLogs[0].CorrelationID
		require.NotEmpty(t, correlationID)
		events := map[string]int{}
		for _, ev := range namespaceLogs {
			require.Equal(t, correlationID, ev.CorrelationID)
			events[ev.Event]++
		}
		require.Equal(t, map[string]int{
			string(auditlog.RatelimitNamespaceDeleteEvent): 1,
			string(auditlog.RatelimitDeleteOverrideEvent):  2,
		}, events)
		for _, id := range overrideIDs {
			require.Len(t, h.FindAuditLogsByTargetID(ctx, t, id), 1)
		}
	})

	t.Run("deletes by name", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.Name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, isDeleted(t, h, ns))
		require.Len(t, h.FindAuditLogsByTargetID(ctx, t, ns.ID), 1)
	})

	t.Run("drops cached entries for the id and name", func(t *testing.T) {
		ns := seedNamespace(t, h, workspace.ID)
		keys := []cache.ScopedKey{
			{WorkspaceID: workspace.ID, Key: ns.ID},
			{WorkspaceID: workspace.ID, Key: ns.Name},
		}
		for _, key := range keys {
			h.Caches.RatelimitNamespace.Set(ctx, key, db.FindRatelimitNamespace{ID: ns.ID, Name: ns.Name})
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		for _, key := range keys {
			_, hit := h.Caches.RatelimitNamespace.Get(ctx, key)
			require.Equal(t, cache.Miss, hit, key.Key)
		}
	})
}
