package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_set_policies"
)

func TestSetPoliciesNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.set_policies")
	headers := authHeaders(rootKey)

	t.Run("nonexistent environment", func(t *testing.T) {
		req := makeRequest(env, []openapi.Policy{firewallPolicy("deny", true)})
		req.Environment = uid.New(uid.EnvironmentPrefix)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("nonexistent project", func(t *testing.T) {
		req := makeRequest(env, []openapi.Policy{firewallPolicy("deny", true)})
		req.Project = uid.New(uid.ProjectPrefix)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("nonexistent app", func(t *testing.T) {
		req := makeRequest(env, []openapi.Policy{firewallPolicy("deny", true)})
		req.App = uid.New(uid.AppPrefix)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("keyauth referencing a nonexistent keyspace", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers,
			makeRequest(env, []openapi.Policy{{
				Name:    "k",
				Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{uid.New(uid.KeySpacePrefix)}},
			}}))
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Type, "key_space_not_found")

		require.Equal(t, "{}", readStoredBlob(t, h, env))
	})

	t.Run("keyauth referencing another workspace's keyspace", func(t *testing.T) {
		other := h.CreateWorkspace()
		foreign := h.CreateApi(seed.CreateApiRequest{WorkspaceID: other.ID})

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers,
			makeRequest(env, []openapi.Policy{{
				Name:    "k",
				Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{foreign.KeyAuthID.String}},
			}}))
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Type, "key_space_not_found")
	})

	t.Run("keyauth referencing a soft-deleted keyspace", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: env.workspaceID})
		_, err := h.DB.RW().ExecContext(context.Background(),
			"UPDATE key_auth SET deleted_at_m = ? WHERE id = ?",
			time.Now().UnixMilli(), api.KeyAuthID.String)
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers,
			makeRequest(env, []openapi.Policy{{
				Name:    "k",
				Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{api.KeyAuthID.String}},
			}}))
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Type, "key_space_not_found")
	})
}
