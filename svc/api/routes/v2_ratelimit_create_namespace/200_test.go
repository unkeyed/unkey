package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_create_namespace"
)

func TestCreateNamespaceSuccess(t *testing.T) {
	ctx := t.Context()
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyNamespace(workspace.ID)))

	t.Run("creates the namespace in the default project", func(t *testing.T) {
		name := newName()

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Name: name})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, strings.HasPrefix(res.Body.Data.Id, "rlns_"), res.Body.Data.Id)
		require.Equal(t, name, res.Body.Data.Name)
		require.NotZero(t, res.Body.Data.CreatedAt)
		require.Zero(t, res.Body.Data.UpdatedAt)

		defaultProjectID, found, err := projects.FindDefaultProject(ctx, h.DB.RO(), workspace.ID)
		require.NoError(t, err)
		require.True(t, found)

		stored, found := findNamespace(t, h, workspace.ID, name)
		require.True(t, found)
		require.Equal(t, res.Body.Data.Id, stored.ID)
		require.Equal(t, defaultProjectID, stored.ProjectID)
		require.Equal(t, res.Body.Data.CreatedAt, stored.CreatedAtM)
		require.False(t, stored.DeletedAtM.Valid)

		logs := h.FindAuditLogsByTargetID(ctx, t, stored.ID)
		require.Len(t, logs, 1)
		require.Equal(t, string(auditlog.RatelimitNamespaceCreateEvent), logs[0].Event)
		require.Equal(t, workspace.ID, logs[0].WorkspaceID)
		require.Len(t, logs[0].Targets, 1)
		require.Equal(t, name, logs[0].Targets[0].Name)
	})

	t.Run("names are case-sensitive", func(t *testing.T) {
		name := newName()
		for _, n := range []string{name, strings.ToUpper(name)} {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Name: n})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, n, res.Body.Data.Name)
		}
	})
}
