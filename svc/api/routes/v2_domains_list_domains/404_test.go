package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

// TestListDomainsIDSlugCollisionsMatchBoth guarantees each optional filter
// independently matches both its ID and slug columns without precedence.
func TestListDomainsIDSlugCollisionsMatchBoth(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	base := seedEnvironment(t, h)
	baseDomain := attachDomain(t, h, base, nil)
	headers := authHeaders(h.CreateRootKey(base.workspaceID, "environment.*.read_domain"))

	projectBySlug := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: base.workspaceID, Name: "Project slug collision", Slug: base.projectID,
	})
	projectApp := h.CreateApp(seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: base.workspaceID, ProjectID: projectBySlug.ID,
		Name: "Project collision app", Slug: randomSlug(), DefaultBranch: "main",
	})
	projectEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: base.workspaceID, ProjectID: projectBySlug.ID,
		AppID: projectApp.ID, Slug: randomSlug(), Description: "Project collision environment",
	})
	projectDomain := attachDomain(t, h, seededEnv{
		workspaceID: base.workspaceID, projectID: projectBySlug.ID, appID: projectApp.ID, environmentID: projectEnvironment.ID,
	}, nil)

	appBySlug := h.CreateApp(seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: base.workspaceID, ProjectID: base.projectID,
		Name: "App slug collision", Slug: base.appID, DefaultBranch: "main",
	})
	appEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: base.workspaceID, ProjectID: base.projectID,
		AppID: appBySlug.ID, Slug: randomSlug(), Description: "App collision environment",
	})
	appDomain := attachDomain(t, h, seededEnv{
		workspaceID: base.workspaceID, projectID: base.projectID, appID: appBySlug.ID, environmentID: appEnvironment.ID,
	}, nil)

	environmentBySlug := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: base.workspaceID, ProjectID: base.projectID,
		AppID: base.appID, Slug: base.environmentID, Description: "Environment slug collision",
	})
	environmentDomain := attachDomain(t, h, seededEnv{
		workspaceID: base.workspaceID, projectID: base.projectID, appID: base.appID, environmentID: environmentBySlug.ID,
	}, nil)

	testCases := []struct {
		name    string
		req     handler.Request
		wantIDs []string
	}{
		{name: "project", req: handler.Request{Project: ptr.P(base.projectID)}, wantIDs: []string{baseDomain.ID, appDomain.ID, environmentDomain.ID, projectDomain.ID}},
		{name: "app", req: handler.Request{App: ptr.P(base.appID)}, wantIDs: []string{baseDomain.ID, appDomain.ID, environmentDomain.ID}},
		{name: "environment", req: handler.Request{Environment: ptr.P(base.environmentID)}, wantIDs: []string{baseDomain.ID, environmentDomain.ID}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tc.req)
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			gotIDs := make([]string, 0, len(res.Body.Data))
			for _, domain := range res.Body.Data {
				gotIDs = append(gotIDs, domain.Id)
			}
			require.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestListDomainsMissingAndMismatchedFiltersReturnEmpty guarantees list filters
// describe a selection, so unresolved or incompatible resources are not errors.
func TestListDomainsMissingAndMismatchedFiltersReturnEmpty(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	domain := attachDomain(t, h, env, nil)
	other := seedEnvironment(t, h)
	headers := authHeaders(h.CreateRootKey(env.workspaceID, "environment.*.read_domain"))
	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing project", req: handler.Request{Project: ptr.P(uid.New(uid.ProjectPrefix))}},
		{name: "missing app", req: handler.Request{App: ptr.P(uid.New(uid.AppPrefix))}},
		{name: "missing environment", req: handler.Request{Environment: ptr.P(uid.New(uid.EnvironmentPrefix))}},
		{name: "app from another project", req: handler.Request{Project: ptr.P(env.projectID), App: ptr.P(other.appID)}},
		{name: "environment from another app", req: handler.Request{App: ptr.P(env.appID), Environment: ptr.P(other.environmentID)}},
		{name: "environment slug mismatched with app", req: handler.Request{App: ptr.P(other.appID), Environment: ptr.P("production"), Search: ptr.P(domain.Domain)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tc.req)
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.Empty(t, res.Body.Data, "%s", res.RawBody)
			require.False(t, res.Body.Pagination.HasMore)
			require.Nil(t, res.Body.Pagination.Cursor)
			require.NotContains(t, res.RawBody, domain.ID)
		})
	}
}

// TestListDomainsCrossWorkspaceFiltersReturnEmpty guarantees globally unique IDs
// and reusable slugs cannot select rows outside the authorized workspace.
func TestListDomainsCrossWorkspaceFiltersReturnEmpty(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	otherWorkspace := h.CreateWorkspace()
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: otherWorkspace.ID, Name: "Other project", Slug: randomSlug(),
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: otherWorkspace.ID, ProjectID: otherProject.ID,
		Name: "Other app", Slug: randomSlug(), DefaultBranch: "main",
	})
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: otherWorkspace.ID, ProjectID: otherProject.ID,
		AppID: otherApp.ID, Slug: "production", Description: "Other environment",
	})
	otherDomain := attachDomain(t, h, seededEnv{
		workspaceID: otherWorkspace.ID, projectID: otherProject.ID, appID: otherApp.ID, environmentID: otherEnvironment.ID,
	}, nil)
	headers := authHeaders(h.CreateRootKey(env.workspaceID, "environment.*.read_domain"))

	requests := []handler.Request{
		{Project: ptr.P(otherProject.ID)},
		{App: ptr.P(otherApp.ID)},
		{Environment: ptr.P(otherEnvironment.ID)},
		{Project: ptr.P(otherProject.Slug), App: ptr.P(otherApp.Slug), Environment: ptr.P(otherEnvironment.Slug)},
	}
	for _, req := range requests {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		require.Empty(t, res.Body.Data, "%s", res.RawBody)
		require.NotContains(t, res.RawBody, otherDomain.ID)
	}
}
