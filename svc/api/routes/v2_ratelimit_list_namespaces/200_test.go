package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_list_namespaces"
)

func TestListNamespacesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	headers := authHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project("*").RatelimitNamespace("*"))))

	t.Run("empty workspace returns empty page", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Empty(t, res.Body.Data)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	live := []seededNamespace{
		seedNamespace(t, h, workspace.ID, "KEBAP."+uid.New("ns")),
		seedNamespace(t, h, workspace.ID, "kebap."+uid.New("ns")),
		seedNamespace(t, h, workspace.ID, "email."+uid.New("ns")),
	}
	newestFirst := []seededNamespace{live[2], live[1], live[0]}
	deleted := seedNamespace(t, h, workspace.ID, uid.New("deleted"))
	require.NoError(t, db.Query.SoftDeleteRatelimitNamespace(t.Context(), h.DB.RW(), db.SoftDeleteRatelimitNamespaceParams{
		Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:  deleted.id,
	}))
	otherWorkspace := seedNamespace(t, h, h.CreateWorkspace().ID, uid.New("other"))

	t.Run("returns live namespaces newest first", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Len(t, res.Body.Data, len(newestFirst), res.RawBody)
		for i, namespace := range res.Body.Data {
			require.Equal(t, newestFirst[i].id, namespace.Id)
			require.Equal(t, newestFirst[i].name, namespace.Name)
			require.Positive(t, namespace.CreatedAt)
			require.Zero(t, namespace.UpdatedAt, "a never-updated namespace omits updatedAt")
		}
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
		require.NotContains(t, res.RawBody, deleted.id)
		require.NotContains(t, res.RawBody, otherWorkspace.id)
	})

	t.Run("pages by id cursor until hasMore is false", func(t *testing.T) {
		var listed []string
		var cursor *string
		for range len(live) + 1 {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: new(1), Cursor: cursor})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Len(t, res.Body.Data, 1, res.RawBody)
			listed = append(listed, res.Body.Data[0].Id)
			if !res.Body.Pagination.HasMore {
				require.Nil(t, res.Body.Pagination.Cursor)
				break
			}
			cursor = res.Body.Pagination.Cursor
			require.Equal(t, newestFirst[len(listed)].id, *cursor)
		}
		require.Equal(t, []string{newestFirst[0].id, newestFirst[1].id, newestFirst[2].id}, listed)
	})

	t.Run("unknown cursor returns empty page", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Cursor: new(uid.New(uid.RatelimitNamespacePrefix))})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Empty(t, res.Body.Data)
		require.False(t, res.Body.Pagination.HasMore)
	})

	t.Run("cursor from another workspace returns empty page", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Cursor: new(otherWorkspace.id)})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Empty(t, res.Body.Data)
		require.False(t, res.Body.Pagination.HasMore)
	})

	t.Run("search matches names case-insensitively", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Search: new("KeBaP")})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, 2, res.RawBody)
		require.Equal(t, live[1].id, res.Body.Data[0].Id)
		require.Equal(t, live[0].id, res.Body.Data[1].Id)
	})

	t.Run("search matches ids", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Search: new(live[1].id)})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, 1, res.RawBody)
		require.Equal(t, live[1].id, res.Body.Data[0].Id)
	})
}

// TestListNamespacesRefillsAuthorizedPages guarantees namespace-scoped grants
// fill each page without exposing denied namespaces.
func TestListNamespacesRefillsAuthorizedPages(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	namespaces := make([]seededNamespace, 6)
	for i := range namespaces {
		namespaces[i] = seedNamespace(t, h, workspace.ID, fmt.Sprintf("api.%d.%s", i, uid.New("ns")))
	}
	allowed := []seededNamespace{namespaces[2], namespaces[4]}
	headers := authHeaders(h.CreateRootKey(workspace.ID,
		fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(allowed[0].projectID).RatelimitNamespace(allowed[0].id)),
		fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(allowed[1].projectID).RatelimitNamespace(allowed[1].id)),
	))

	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: new(1)})
	require.Equal(t, http.StatusOK, first.Status, "expected 200, received: %s", first.RawBody)
	require.Len(t, first.Body.Data, 1, first.RawBody)
	require.Equal(t, allowed[1].id, first.Body.Data[0].Id)
	require.True(t, first.Body.Pagination.HasMore)
	require.Equal(t, new(allowed[0].id), first.Body.Pagination.Cursor)

	last := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: new(1), Cursor: first.Body.Pagination.Cursor})
	require.Equal(t, http.StatusOK, last.Status, "expected 200, received: %s", last.RawBody)
	require.Len(t, last.Body.Data, 1, last.RawBody)
	require.Equal(t, allowed[0].id, last.Body.Data[0].Id)
	require.False(t, last.Body.Pagination.HasMore)

	for _, i := range []int{0, 1, 3, 5} {
		require.NotContains(t, first.RawBody, namespaces[i].id)
		require.NotContains(t, last.RawBody, namespaces[i].id)
	}
}
