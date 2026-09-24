package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
)

// TestGetApiAuthorizesExactKeyspaceReadURN guarantees that URN keyspace
// read access is sufficient without a legacy API permission.
func TestGetApiAuthorizesExactKeyspaceReadURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Caches: h.Caches}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	permission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s#read",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
	)
	rootKey := h.CreateRootKey(workspace.ID, permission)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](
		h,
		route,
		headers,
		handler.Request{ApiId: api.ID},
	)

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, api.ID, res.Body.Data.Id)
}

// TestGetApiRejectsURNOutsideExactKeyspaceReadScope guarantees that every
// URN resource dimension and the action restrict access.
func TestGetApiRejectsURNOutsideExactKeyspaceReadScope(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Caches: h.Caches}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	apiProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "API project",
		Slug:        uid.New("project"),
	})
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE apis SET project_id = ? WHERE id = ?", apiProject.ID, api.ID)
	require.NoError(t, err)
	testCases := map[string]string{
		"wrong keyspace": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/keyspaces/%s#read",
			workspace.ID,
			api.ProjectID,
			uid.New(uid.KeySpacePrefix),
		),
		"wrong project": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/keyspaces/%s#read",
			workspace.ID,
			apiProject.ID,
			api.KeyAuthID.String,
		),
		"wrong workspace": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/keyspaces/%s#read",
			uid.New(uid.WorkspacePrefix),
			api.ProjectID,
			api.KeyAuthID.String,
		),
		"wrong action": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/keyspaces/%s#write",
			workspace.ID,
			api.ProjectID,
			api.KeyAuthID.String,
		),
	}

	for name, permission := range testCases {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, permission)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				headers,
				handler.Request{ApiId: api.ID},
			)

			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/data/api_not_found", res.Body.Error.Type)
		})
	}
}

// TestGetApiAuthorizesWildcardURNs guarantees that URN wildcard permissions
// cover the exact keyspace resource required by the route.
func TestGetApiAuthorizesWildcardURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Caches: h.Caches}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	testCases := map[string]string{
		"keyspace wildcard": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/keyspaces/*#read",
			workspace.ID,
			api.ProjectID,
		),
		"project and keyspace wildcards": fmt.Sprintf(
			"unkey:v1:%s:projects/*/keyspaces/*#read",
			workspace.ID,
		),
		"project descendants": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/**#read",
			workspace.ID,
			api.ProjectID,
		),
		"workspace admin": fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
	}

	for name, permission := range testCases {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, permission)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](
				h,
				route,
				headers,
				handler.Request{ApiId: api.ID},
			)

			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, api.ID, res.Body.Data.Id)
		})
	}
}
