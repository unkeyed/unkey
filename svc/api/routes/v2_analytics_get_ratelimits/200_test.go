package handler

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test200_RawEventsWildcard(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	namespaceID := createNamespace(t, h, workspaceID)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	h.RatelimitEvents.Buffer(schema.Ratelimit{RequestID: uid.New(uid.RequestPrefix), Time: time.Now().UnixMilli(), WorkspaceID: workspaceID, NamespaceID: namespaceID, Identifier: "user", Passed: true, Limit: 10, Remaining: 9, Tokens: 1})
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT namespace_id, identifier FROM ratelimits_v1 WHERE namespace_id = '%s'", namespaceID)})
		require.Equal(c, 200, res.Status)
		require.Len(c, res.Body.Data, 1)
		require.Equal(c, namespaceID, res.Body.Data[0]["namespace_id"])
		require.Equal(c, "user", res.Body.Data[0]["identifier"])
	}, 30*time.Second, time.Second)
}

func Test200_QualifiedNamespacePredicate(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	namespaceID := createNamespace(t, h, workspaceID)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT count(*) AS count FROM ratelimits_v1 AS r WHERE r.namespace_id = '%s'", namespaceID)})
	require.Equal(t, 200, res.Status)
}

func Test200_WildcardUnionCannotEscapeNamedNamespace(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	allowed, other := createNamespace(t, h, workspaceID), createNamespace(t, h, workspaceID)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	for _, id := range []string{allowed, other} {
		h.RatelimitEvents.Buffer(schema.Ratelimit{RequestID: uid.New(uid.RequestPrefix), Time: time.Now().UnixMilli(), WorkspaceID: workspaceID, NamespaceID: id, Identifier: id, Passed: true, Limit: 10})
	}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		query := fmt.Sprintf("SELECT namespace_id FROM ratelimits_v1 WHERE namespace_id = '%s' UNION ALL SELECT namespace_id FROM ratelimits_v1", allowed)
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
		require.Equal(c, 200, res.Status)
		require.NotEmpty(c, res.Body.Data)
		for _, row := range res.Body.Data {
			require.Equal(c, allowed, row["namespace_id"])
		}
	}, 30*time.Second, time.Second)
}

func Test200_ScopedPermissionCannotEscapeWithOr(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	allowed := createNamespace(t, h, workspaceID)
	other := createNamespace(t, h, workspaceID)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit."+allowed+".read_analytics")
	for _, id := range []string{allowed, other} {
		h.RatelimitEvents.Buffer(schema.Ratelimit{RequestID: uid.New(uid.RequestPrefix), Time: time.Now().UnixMilli(), WorkspaceID: workspaceID, NamespaceID: id, Identifier: id, Passed: true, Limit: 10})
	}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT namespace_id FROM ratelimits_v1 WHERE namespace_id = '%s' OR 1=1", allowed)})
		require.Equal(c, 200, res.Status)
		require.Equal(c, []map[string]any{{"namespace_id": allowed}}, res.Body.Data)
	}, 30*time.Second, time.Second)
}

func Test200_ExactPermissionsForMultipleNamespaces(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	first, second := createNamespace(t, h, workspaceID), createNamespace(t, h, workspaceID)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit."+first+".read_analytics", "ratelimit."+second+".read_analytics")
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT count(*) AS count FROM ratelimits_v1 WHERE namespace_id IN ('%s','%s')", first, second)})
	require.Equal(t, 200, res.Status)
}

func Test200_SoftDeletedNamespace(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	id := createNamespace(t, h, workspaceID)
	require.NoError(t, db.Query.SoftDeleteRatelimitNamespace(context.Background(), h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{ID: id, Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}}))
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT count(*) FROM ratelimits_v1 WHERE namespace_id = '%s'", id)})
	require.Equal(t, 200, res.Status)
}
