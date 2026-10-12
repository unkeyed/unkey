package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/flags"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_flags"
)

func TestGetFlagsResolvesOnlyCallerOverrides(t *testing.T) {
	h := testutil.NewHarness(t)
	route, database := newRoute(t, h)
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	otherWorkspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#read")

	for _, flag := range []db.UpsertFlagParams{
		{Slug: "default_on", Description: "Enabled by default", DefaultValue: true, AllowEnable: false, AllowDisable: true},
		{Slug: "default_off", Description: "Disabled by default", DefaultValue: false, AllowEnable: true, AllowDisable: false},
		{Slug: "override_off", Description: "Explicit opt out", DefaultValue: true, AllowEnable: true, AllowDisable: true},
		{Slug: "override_on", Description: "Explicit opt in", DefaultValue: false, AllowEnable: true, AllowDisable: false},
	} {
		require.NoError(t, db.Query.UpsertFlag(t.Context(), database.RW(), flag))
	}
	for _, override := range []db.UpsertWorkspaceFlagOverrideParams{
		{WorkspaceID: workspaceID, FlagSlug: "override_off", Value: false},
		{WorkspaceID: workspaceID, FlagSlug: "override_on", Value: true},
		{WorkspaceID: otherWorkspace.ID, FlagSlug: "default_on", Value: false},
		{WorkspaceID: otherWorkspace.ID, FlagSlug: "default_off", Value: true},
		{WorkspaceID: otherWorkspace.ID, FlagSlug: "override_off", Value: true},
		{WorkspaceID: otherWorkspace.ID, FlagSlug: "override_on", Value: false},
	} {
		require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), database.RW(), override))
	}

	res := callGetFlags(h, route, rootKey)
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, map[string]openapi.V2WorkspaceGetFlagsFlag{
		"default_on":   {Value: true, Description: "Enabled by default", AllowEnable: false, AllowDisable: true},
		"default_off":  {Value: false, Description: "Disabled by default", AllowEnable: true, AllowDisable: false},
		"override_off": {Value: false, Description: "Explicit opt out", AllowEnable: true, AllowDisable: true},
		"override_on":  {Value: true, Description: "Explicit opt in", AllowEnable: true, AllowDisable: false},
	}, res.Body.Data)

	type workspaceSelector struct {
		WorkspaceID string `json:"workspaceId"`
	}
	attempt := testutil.CallRoute[workspaceSelector, handler.Response](h, route,
		http.Header{"Authorization": {"Bearer " + rootKey}},
		workspaceSelector{WorkspaceID: otherWorkspace.ID},
	)
	require.Equal(t, http.StatusOK, attempt.Status, attempt.RawBody)
	require.Equal(t, res.Body.Data, attempt.Body.Data)

	otherKey := h.CreateRootKey(otherWorkspace.ID, "unkey:v1:"+otherWorkspace.ID+":flags#read")
	other := callGetFlags(h, route, otherKey)
	require.Equal(t, http.StatusOK, other.Status, other.RawBody)
	require.True(t, other.Body.Data["override_off"].Value)
	require.False(t, other.Body.Data["override_on"].Value)

	require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), database.RW(), db.UpsertWorkspaceFlagOverrideParams{
		WorkspaceID: workspaceID, FlagSlug: "override_off", Value: true,
	}))
	cached := callGetFlags(h, route, rootKey)
	require.Equal(t, http.StatusOK, cached.Status, cached.RawBody)
	require.Equal(t, res.Body.Data, cached.Body.Data)

	h.Clock.Tick(time.Minute)
	refreshed := callGetFlags(h, route, rootKey)
	require.Equal(t, http.StatusOK, refreshed.Status, refreshed.RawBody)
	require.True(t, refreshed.Body.Data["override_off"].Value)

	require.NoError(t, database.Close())
	cached = callGetFlags(h, route, rootKey)
	require.Equal(t, http.StatusOK, cached.Status, cached.RawBody)
	require.Equal(t, refreshed.Body.Data, cached.Body.Data)
	denied := callGetFlags(h, route, h.CreateRootKey(workspaceID))
	require.Equal(t, http.StatusForbidden, denied.Status, denied.RawBody)
}

func TestGetFlagsEmpty(t *testing.T) {
	h := testutil.NewHarness(t)
	route, database := newRoute(t, h)
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#read")

	res := callGetFlags(h, route, rootKey)
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.NotNil(t, res.Body.Data)
	require.Empty(t, res.Body.Data)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	require.NoError(t, database.Close())
	cached := callGetFlags(h, route, rootKey)
	require.Equal(t, http.StatusOK, cached.Status, cached.RawBody)
	require.NotNil(t, cached.Body.Data)
	require.Empty(t, cached.Body.Data)
}

func TestGetFlagsDatabaseFailure(t *testing.T) {
	h := testutil.NewHarness(t)
	route, database := newRoute(t, h)
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":flags#read")
	require.NoError(t, database.Close())

	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header.Set("Authorization", "Bearer "+rootKey)
	res := testutil.CallRaw[openapi.InternalServerErrorResponse](h, req)
	require.Equal(t, http.StatusInternalServerError, res.Status, res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/service_unavailable", res.Body.Error.Type)
	require.Equal(t, "Failed to read the workspace's flags.", res.Body.Error.Detail)
}

func newRoute(t *testing.T, h *testutil.Harness) (*handler.Handler, db.Database) {
	t.Helper()
	config := containers.MySQLIsolated(t)
	database, err := db.New(db.Config{PrimaryDSN: config.DSN, ReadOnlyDSN: "", Tags: sqlcomment.Disabled()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	service, err := flags.New(flags.Config{DB: db.ToMySQL(database), Clock: h.Clock})
	require.NoError(t, err)
	t.Cleanup(service.Close)
	return &handler.Handler{Flags: service}, database
}

func callGetFlags(h *testutil.Harness, route *handler.Handler, rootKey string) testutil.TestResponse[handler.Response] {
	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	if rootKey != "" {
		req.Header.Set("Authorization", "Bearer "+rootKey)
	}
	return testutil.CallRaw[handler.Response](h, req)
}
