package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_list_policies"
)

func TestListPoliciesNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/policies/*#read", env.workspaceID, env.projectID, env.appID, env.environmentID))
	headers := authHeaders(rootKey)

	t.Run("nonexistent environment", func(t *testing.T) {
		req := makeRequest(env)
		req.Environment = uid.New(uid.EnvironmentPrefix)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("nonexistent project", func(t *testing.T) {
		req := makeRequest(env)
		req.Project = uid.New(uid.ProjectPrefix)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("nonexistent app", func(t *testing.T) {
		req := makeRequest(env)
		req.App = uid.New(uid.AppPrefix)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("another workspace's environment", func(t *testing.T) {
		other := h.CreateWorkspace()
		foreignKey := h.CreateRootKey(other.ID, fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/gateway/policies/*#read", other.ID))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(foreignKey), makeRequest(env))
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})
}
