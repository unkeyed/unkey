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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environment_variables"
)

func TestListEnvironmentVariablesEnvironmentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Vault: h.Vault}
	h.Register(route)

	env := seedEnvironment(t, h)
	missingID := uid.New(uid.EnvironmentPrefix)
	rootKey := h.CreateRootKey(env.workspaceID, rbac.U(urn.New().Workspace(env.workspaceID).Project(env.projectID).App(env.appID).Environment(missingID).Variable("*"), permissions.Read).Value)
	headers := authHeaders(rootKey)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: missingID,
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}
