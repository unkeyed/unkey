package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_get_domain"
)

// TestGetDomainPermissions is a permission matrix. Rejections are 404, not 403: a
// caller that may not read the domain's environment must not be able to tell a real
// domain from a missing one by the status code alone.
func TestGetDomainPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.read_domain"}, shouldPass: true},
		{name: "specific environment permission", permissions: []string{fmt.Sprintf("environment.%s.read_domain", seeded.environmentID)}, shouldPass: true},
		{name: "canonical urn grant", permissions: []string{fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/domains/*#read_domain", seeded.workspaceID, seeded.projectID, seeded.appID, seeded.environmentID)}, shouldPass: true},
		{name: "permission alongside unrelated grants", permissions: []string{"api.*.read_api", "environment.*.read_domain"}, shouldPass: true},
		{name: "create action is not enough", permissions: []string{"environment.*.create_domain"}, shouldPass: false},
		{name: "read_environment is not enough", permissions: []string{"environment.*.read_environment"}, shouldPass: false},
		{name: "adjacent read action is not enough", permissions: []string{"environment.*.read_environment_variables"}, shouldPass: false},
		{name: "read_policies is not enough", permissions: []string{"environment.*.read_policies"}, shouldPass: false},
		{name: "action scoped to the wrong resource type", permissions: []string{"app.*.read_domain"}, shouldPass: false},
		{name: "parent-scoped grant does not cascade", permissions: []string{fmt.Sprintf("app.%s.read_app", seeded.appID)}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.read_domain", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "urn missing the project and app segments", permissions: []string{fmt.Sprintf("unkey:v1:%s:environments/*#read_domain", seeded.workspaceID)}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(seeded.workspaceID, tc.permissions...)

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
				Domain: seeded.domain,
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200 for %v, received: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, received: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, seeded.domainID, "masked 404 leaked the domain id: %s", res.RawBody)
			require.NotContains(t, res.RawBody, seeded.environmentID, "masked 404 leaked the environment id: %s", res.RawBody)
		})
	}
}

// TestGetDomainExistenceNotLeaked asserts that a zero-permission root key targeting a
// real domain and a nonexistent one receives responses that are indistinguishable
// apart from the request id. Without this, permission rejections would act as an
// existence oracle over every domain in the workspace, and domain names are guessable
// in a way ids are not. Do not weaken this by returning 403 for the real domain.
func TestGetDomainExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID)
	headers := authHeaders(rootKey)

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Domain: seeded.domain,
	})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Domain: randomDomain(),
	})

	require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotContains(t, realRes.RawBody, seeded.domainID)
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
	require.Equal(t, missingRes.Body.Error.Title, realRes.Body.Error.Title)
}

// TestGetDomainExistenceNotLeakedWithPartialGrant covers the subtler oracle: a key
// that may read one environment's domains must not learn about another environment's.
func TestGetDomainExistenceNotLeakedWithPartialGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	granted := seedDomain(t, h, nil)
	other := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(granted.workspaceID, fmt.Sprintf("environment.%s.read_domain", granted.environmentID))
	headers := authHeaders(rootKey)

	otherRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Domain: other.domain,
	})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Domain: randomDomain(),
	})

	require.Equal(t, http.StatusNotFound, otherRes.Status, "expected 404, received: %s", otherRes.RawBody)
	require.NotContains(t, otherRes.RawBody, other.domainID)
	require.NotContains(t, otherRes.RawBody, other.environmentID)
	require.Equal(t, missingRes.Body.Error.Detail, otherRes.Body.Error.Detail)
	require.Equal(t, missingRes.Body.Error.Type, otherRes.Body.Error.Type)
	require.Equal(t, missingRes.Body.Error.Status, otherRes.Body.Error.Status)
}
