package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

// TestListDomainsPermissions is a permission matrix. Rejections are 404, not 403: a
// caller that may not read the environment must not be able to tell a real one from a
// missing one by the status code alone.
func TestListDomainsPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.read_domain"}, shouldPass: true},
		{name: "specific environment permission", permissions: []string{fmt.Sprintf("environment.%s.read_domain", env.environmentID)}, shouldPass: true},
		{name: "canonical urn grant", permissions: []string{fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/domains/*#read", env.workspaceID, env.projectID, env.appID, env.environmentID)}, shouldPass: true},
		{name: "permission alongside unrelated grants", permissions: []string{"api.*.read_api", "environment.*.read_domain"}, shouldPass: true},
		{name: "create action is not enough", permissions: []string{"environment.*.create_domain"}, shouldPass: false},
		{name: "read_environment is not enough", permissions: []string{"environment.*.read_environment"}, shouldPass: false},
		{name: "adjacent read action is not enough", permissions: []string{"environment.*.read_environment_variables"}, shouldPass: false},
		{name: "read_policies is not enough", permissions: []string{"environment.*.read_policies"}, shouldPass: false},
		{name: "action scoped to the wrong resource type", permissions: []string{"app.*.read_domain"}, shouldPass: false},
		{name: "parent-scoped grant does not cascade", permissions: []string{fmt.Sprintf("app.%s.read_app", env.appID)}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.read_domain", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "urn missing the project and app segments", permissions: []string{fmt.Sprintf("unkey:v1:%s:environments/*#read", env.workspaceID)}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(env.workspaceID, tc.permissions...)

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), makeRequest(env))
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200 for %v, received: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, received: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, env.environmentID, "masked 404 leaked the environment id: %s", res.RawBody)
		})
	}
}

// TestListDomainsBroadScopesRequireWildcard guarantees that a key scoped to one
// environment cannot list domains from a parent scope that it may not read.
func TestListDomainsBroadScopesRequireWildcard(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID, "environment."+env.environmentID+".read_domain")

	testCases := []struct {
		name           string
		req            handler.Request
		expectedStatus int
	}{
		{name: "workspace", req: handler.Request{}, expectedStatus: http.StatusForbidden},
		{name: "project", req: handler.Request{Project: ptr.P(env.projectID)}, expectedStatus: http.StatusNotFound},
		{name: "app", req: handler.Request{Project: ptr.P(env.projectID), App: ptr.P(env.appID)}, expectedStatus: http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, authHeaders(rootKey), tc.req)
			require.Equal(t, tc.expectedStatus, res.Status, "unexpected status, received: %s", res.RawBody)
		})
	}
}

// TestListDomainsProjectScopeAcceptsCanonicalPermission guarantees that dashboard
// roles can list all domains in a project with their canonical domain permission.
func TestListDomainsProjectScopeAcceptsCanonicalPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(
		env.workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/domains/*#read", env.workspaceID),
	)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project: ptr.P(env.projectID),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
}

// TestListDomainsParentExistenceNotLeaked guarantees that scope resolution does
// not let an unauthorized caller distinguish a real parent from a missing one.
func TestListDomainsParentExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID)
	headers := authHeaders(rootKey)

	testCases := []struct {
		name       string
		realReq    handler.Request
		missingReq handler.Request
	}{
		{
			name:       "project",
			realReq:    handler.Request{Project: ptr.P(env.projectID)},
			missingReq: handler.Request{Project: ptr.P(uid.New(uid.ProjectPrefix))},
		},
		{
			name:       "app",
			realReq:    handler.Request{Project: ptr.P(env.projectID), App: ptr.P(env.appID)},
			missingReq: handler.Request{Project: ptr.P(env.projectID), App: ptr.P(uid.New(uid.AppPrefix))},
		},
		{
			name:       "app with missing project",
			realReq:    handler.Request{Project: ptr.P(env.projectID), App: ptr.P(env.appID)},
			missingReq: handler.Request{Project: ptr.P(uid.New(uid.ProjectPrefix)), App: ptr.P(env.appID)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, tc.realReq)
			missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, tc.missingReq)

			require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
			require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
			require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
			require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
			require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
			require.Equal(t, missingRes.Body.Error.Title, realRes.Body.Error.Title)
		})
	}
}

// TestListDomainsExistenceNotLeaked asserts that a zero-permission root key targeting a
// real environment and a nonexistent one receives responses that are indistinguishable
// apart from the request id. Without this, permission rejections would act as an
// existence oracle over every environment in the workspace. Do not weaken this by
// returning 403 for the real environment.
func TestListDomainsExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	attachDomain(t, h, env, nil)
	rootKey := h.CreateRootKey(env.workspaceID)
	headers := authHeaders(rootKey)

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, makeRequest(env))
	missingRequests := []struct {
		name string
		req  handler.Request
	}{
		{
			name: "environment",
			req: handler.Request{
				Project:     ptr.P(env.projectID),
				App:         ptr.P(env.appID),
				Environment: ptr.P(uid.New(uid.EnvironmentPrefix)),
			},
		},
		{
			name: "app",
			req: handler.Request{
				Project:     ptr.P(env.projectID),
				App:         ptr.P(uid.New(uid.AppPrefix)),
				Environment: ptr.P(env.environmentID),
			},
		},
		{
			name: "project",
			req: handler.Request{
				Project:     ptr.P(uid.New(uid.ProjectPrefix)),
				App:         ptr.P(env.appID),
				Environment: ptr.P(env.environmentID),
			},
		},
	}

	for _, tc := range missingRequests {
		t.Run(tc.name, func(t *testing.T) {
			missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, tc.req)

			require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
			require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
			require.NotContains(t, realRes.RawBody, env.environmentID)
			require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
			require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
			require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
			require.Equal(t, missingRes.Body.Error.Title, realRes.Body.Error.Title)
		})
	}
}
