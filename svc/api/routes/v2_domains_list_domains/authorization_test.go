package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

// TestListDomainsOmitsUnauthorizedRows guarantees filters
// cannot turn an unrelated or empty permission set into domain-read access.
func TestListDomainsOmitsUnauthorizedRows(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	domain := attachDomain(t, h, env, nil)
	testCases := []struct {
		name        string
		permissions []string
		req         handler.Request
	}{
		{name: "no permissions unfiltered", permissions: nil, req: handler.Request{}},
		{name: "no permissions filtered", permissions: nil, req: makeRequest(env)},
		{name: "create action", permissions: []string{"environment.*.create_domain"}, req: makeRequest(env)},
		{name: "unrelated legacy permission", permissions: []string{"api.*.read_api"}, req: handler.Request{Environment: ptr.P(env.environmentID)}},
		{
			name: "canonical write action",
			permissions: []string{rbac.U(
				urn.New().Workspace(env.workspaceID).Project(env.projectID).App(env.appID).Environment(env.environmentID).Domain(domain.ID),
				permissions.Write,
			).Value},
			req: handler.Request{Search: ptr.P(domain.Domain)},
		},
		{
			name: "canonical read in another workspace",
			permissions: []string{rbac.U(
				urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(env.projectID).App(env.appID).Environment(env.environmentID).Domain(domain.ID),
				permissions.Read,
			).Value},
			req: makeRequest(env),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(env.workspaceID, tc.permissions...)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), tc.req)
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.Empty(t, res.Body.Data)
			require.False(t, res.Body.Pagination.HasMore)
			require.Nil(t, res.Body.Pagination.Cursor)
			require.NotContains(t, res.RawBody, domain.ID)
		})
	}
}

// TestListDomainsAcceptsDomainReadGrants preserves legacy wildcard and canonical row access.
func TestListDomainsAcceptsDomainReadGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	domain := attachDomain(t, h, env, nil)
	testCases := []struct {
		name  string
		grant string
	}{
		{name: "legacy environment wildcard", grant: "environment.*.read_domain"},
		{
			name: "narrow canonical domain",
			grant: rbac.U(
				urn.New().Workspace(env.workspaceID).Project(env.projectID).App(env.appID).Environment(env.environmentID).Domain(domain.ID),
				permissions.Read,
			).Value,
		},
		{
			name:  "canonical descendant wildcard",
			grant: urn.New().Workspace(env.workspaceID).Project(env.projectID).String() + "/**#read",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(h.CreateRootKey(env.workspaceID, tc.grant)), handler.Request{})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Len(t, res.Body.Data, 1)
			require.Equal(t, domain.ID, res.Body.Data[0].Id)
		})
	}
}
