package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_remove_environment_variables"
)

func TestRemoveEnvironmentVariablesEnvironmentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, rbac.U(urn.New().Workspace(env.workspaceID).Project("*").App("*").Environment("*").Variable("*"), permissions.Delete).Value)
	headers := authHeaders(rootKey)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: uid.New(uid.EnvironmentPrefix),
		Variables:   []string{"KEY"},
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}
