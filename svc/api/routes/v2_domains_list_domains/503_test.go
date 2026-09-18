package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

// TestListDomainsScanBudget never returns a partial page or a denied cursor when
// scanning stops, but accepts exhaustion and authorized lookahead at the boundary.
func TestListDomainsScanBudget(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	env := seedEnvironment(t, h)
	prefix := uid.DNS1035(16)
	for i := 0; i < 10_000; i++ {
		attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
			req.ID = fmt.Sprintf("dom_%s_%05d", prefix, i)
		})
	}
	firstID := fmt.Sprintf("dom_%s_00000", prefix)
	lastID := fmt.Sprintf("dom_%s_09999", prefix)
	resource := urn.New().Workspace(env.workspaceID).Project(env.projectID).App(env.appID).Environment(env.environmentID)
	grant := rbac.U(resource.Domain(firstID), permissions.Read).Value
	headers := authHeaders(h.CreateRootKey(env.workspaceID, grant))
	req := handler.Request{Limit: ptr.P(1)}
	exhausted := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, exhausted.Status, "%s", exhausted.RawBody)
	require.Len(t, exhausted.Body.Data, 1)
	require.False(t, exhausted.Body.Pagination.HasMore)
	require.Nil(t, exhausted.Body.Pagination.Cursor)

	attachDomain(t, h, env, func(req *seed.CreateCustomDomainRequest) {
		req.ID = fmt.Sprintf("dom_%s_10000", prefix)
	})
	limited := testutil.CallRoute[handler.Request, openapi.ServiceUnavailableErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusServiceUnavailable, limited.Status, "%s", limited.RawBody)
	require.NotContains(t, limited.RawBody, firstID)
	require.NotContains(t, limited.RawBody, "pagination")
	require.NotContains(t, limited.RawBody, `"data"`)

	boundaryHeaders := authHeaders(h.CreateRootKey(env.workspaceID, grant, rbac.U(resource.Domain(lastID), permissions.Read).Value))
	boundary := testutil.CallRoute[handler.Request, handler.Response](h, route, boundaryHeaders, req)
	require.Equal(t, http.StatusOK, boundary.Status, "%s", boundary.RawBody)
	require.Len(t, boundary.Body.Data, 1)
	require.Equal(t, firstID, boundary.Body.Data[0].Id)
	require.Equal(t, ptr.P(lastID), boundary.Body.Pagination.Cursor)
	require.True(t, boundary.Body.Pagination.HasMore)
}
