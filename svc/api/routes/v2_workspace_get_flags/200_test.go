package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/flags"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_flags"
)

func newRoute(t *testing.T, h *testutil.Harness, database db.Database) *handler.Handler {
	t.Helper()
	service, err := flags.New(flags.Config{DB: db.ToMySQL(database), Clock: h.Clock})
	require.NoError(t, err)
	t.Cleanup(service.Close)
	return &handler.Handler{Flags: service}
}

func callGetFlags(h *testutil.Harness, route *handler.Handler, headers http.Header) testutil.TestResponse[handler.Response] {
	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header = headers
	return testutil.CallRaw[handler.Response](h, req)
}

func bearer(rootKey string) http.Header {
	return http.Header{"Authorization": {"Bearer " + rootKey}}
}

func insertFlag(t *testing.T, h *testutil.Harness, defaultValue bool) string {
	t.Helper()
	slug := uid.New(uid.TestPrefix)
	require.NoError(t, db.Query.UpsertFlag(t.Context(), h.DB.RW(), db.UpsertFlagParams{
		Slug:         slug,
		Description:  t.Name(),
		DefaultValue: defaultValue,
		Visible:      false,
		AllowEnable:  false,
		AllowDisable: false,
	}))
	return slug
}

func setOverride(t *testing.T, h *testutil.Harness, workspaceID, slug string, value bool) {
	t.Helper()
	require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), h.DB.RW(), db.UpsertWorkspaceFlagOverrideParams{
		WorkspaceID: workspaceID,
		FlagSlug:    slug,
		Value:       value,
	}))
}

// TestGetFlagsReturnsResolvedValues guarantees that the response contains every
// defined flag with the caller's workspace override applied, keeps flags that
// resolve to false, and ignores overrides of other workspaces.
func TestGetFlagsReturnsResolvedValues(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(t, h, h.DB)
	h.Register(route)

	workspace := h.CreateWorkspace()
	otherWorkspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:flags#read", workspace.ID))

	defaultOn := insertFlag(t, h, true)
	defaultOff := insertFlag(t, h, false)
	disabled := insertFlag(t, h, true)
	setOverride(t, h, workspace.ID, disabled, false)
	enabled := insertFlag(t, h, false)
	setOverride(t, h, workspace.ID, enabled, true)
	setOverride(t, h, otherWorkspace.ID, defaultOff, true)

	res := callGetFlags(h, route, bearer(rootKey))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	got := res.Body.Data.Flags
	for slug, want := range map[string]bool{defaultOn: true, defaultOff: false, disabled: false, enabled: true} {
		value, ok := got[slug]
		require.True(t, ok, "flag %s missing from %v", slug, got)
		require.Equal(t, want, value, "flag %s", slug)
	}
}
