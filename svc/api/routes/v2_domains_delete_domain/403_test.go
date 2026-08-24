package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_delete_domain"
)

// TestDeleteDomainPermissions is a permission matrix. Rejections are 404, not 403: a
// caller that may not delete the domain's environment must not be able to tell a real
// domain from a missing one by the status code alone.
func TestDeleteDomainPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.delete_domain"}, shouldPass: true},
		{name: "specific environment permission", permissions: []string{"environment.<env>.delete_domain"}, shouldPass: true},
		{name: "canonical urn grant", permissions: []string{"<urn>.delete_domain"}, shouldPass: true},
		{name: "permission alongside unrelated grants", permissions: []string{"api.*.read_api", "environment.*.delete_domain"}, shouldPass: true},
		{name: "create action is not enough", permissions: []string{"environment.*.create_domain"}, shouldPass: false},
		{name: "read action is not enough", permissions: []string{"environment.*.read_domain"}, shouldPass: false},
		{name: "adjacent delete action is not enough", permissions: []string{"environment.*.remove_environment_variables"}, shouldPass: false},
		{name: "action scoped to the wrong resource type", permissions: []string{"app.*.delete_domain"}, shouldPass: false},
		{name: "app delete does not cascade", permissions: []string{"app.*.delete_app"}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.delete_domain", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seeded := seedDomain(t, h, nil)
			permissions := make([]string, len(tc.permissions))
			for i, p := range tc.permissions {
				switch p {
				case "environment.<env>.delete_domain":
					p = fmt.Sprintf("environment.%s.delete_domain", seeded.environmentID)
				case "<urn>.delete_domain":
					p = fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s#delete_domain", seeded.workspaceID, seeded.projectID, seeded.appID, seeded.environmentID)
				}
				permissions[i] = p
			}
			rootKey := h.CreateRootKey(seeded.workspaceID, permissions...)
			callsBefore := len(ctrlClient.DeleteCustomDomainCalls)

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
				Domain: seeded.domain,
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200 for %v, received: %s", permissions, res.RawBody)
				require.Len(t, ctrlClient.DeleteCustomDomainCalls, callsBefore+1)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, received: %s", permissions, res.RawBody)
			require.NotContains(t, res.RawBody, seeded.domainID, "masked 404 leaked the domain id: %s", res.RawBody)
			require.NotContains(t, res.RawBody, seeded.environmentID, "masked 404 leaked the environment id: %s", res.RawBody)
			require.Len(t, ctrlClient.DeleteCustomDomainCalls, callsBefore,
				"a rejected request must not delete anything")
		})
	}
}

// TestDeleteDomainExistenceNotLeaked asserts that a zero-permission root key targeting
// a real domain and a nonexistent one receives responses that are indistinguishable
// apart from the request id. Without this, permission rejections would act as an
// existence oracle over every domain in the workspace, and a delete endpoint makes an
// especially attractive probe because names are guessable. Do not weaken this by
// returning 403 for the real domain.
func TestDeleteDomainExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
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
	require.Empty(t, ctrlClient.DeleteCustomDomainCalls, "neither probe may reach ctrl")
}

// TestDeleteDomainExistenceNotLeakedWithPartialGrant covers the subtler oracle: a key
// that may delete one environment's domains must not learn about another environment's.
func TestDeleteDomainExistenceNotLeakedWithPartialGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	granted := seedDomain(t, h, nil)
	other := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(granted.workspaceID, fmt.Sprintf("environment.%s.delete_domain", granted.environmentID))
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
	require.Empty(t, ctrlClient.DeleteCustomDomainCalls, "the ungranted domain must stay untouched")
}
