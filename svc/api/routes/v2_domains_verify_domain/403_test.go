package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_verify_domain"
)

// TestVerifyDomainPermissions is a permission matrix. Rejections are 404, not 403: a
// caller that may not verify the domain's environment must not be able to tell a real
// domain from a missing one by the status code alone.
func TestVerifyDomainPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.verify_domain"}, shouldPass: true},
		{name: "specific environment permission", permissions: []string{"environment.<env>.verify_domain"}, shouldPass: true},
		{name: "permission alongside unrelated grants", permissions: []string{"api.*.read_api", "environment.*.verify_domain"}, shouldPass: true},
		{name: "create action is not enough", permissions: []string{"environment.*.create_domain"}, shouldPass: false},
		{name: "read action is not enough", permissions: []string{"environment.*.read_domain"}, shouldPass: false},
		{name: "delete action is not enough", permissions: []string{"environment.*.delete_domain"}, shouldPass: false},
		{name: "adjacent environment action is not enough", permissions: []string{"environment.*.set_environment_variables"}, shouldPass: false},
		{name: "action scoped to the wrong resource type", permissions: []string{"app.*.verify_domain"}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.verify_domain", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seeded := seedDomain(t, h, nil)
			permissions := make([]string, len(tc.permissions))
			for i, p := range tc.permissions {
				if p == "environment.<env>.verify_domain" {
					p = fmt.Sprintf("environment.%s.verify_domain", seeded.environmentID)
				}
				permissions[i] = p
			}
			rootKey := h.CreateRootKey(seeded.workspaceID, permissions...)
			callsBefore := len(ctrlClient.RetryVerificationCalls)

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
				Domain: seeded.domain,
			})
			if tc.shouldPass {
				require.Equal(t, http.StatusAccepted, res.Status, "expected 202 for %v, received: %s", permissions, res.RawBody)
				require.Len(t, ctrlClient.RetryVerificationCalls, callsBefore+1)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, received: %s", permissions, res.RawBody)
			require.NotContains(t, res.RawBody, seeded.domainID, "masked 404 leaked the domain id: %s", res.RawBody)
			require.NotContains(t, res.RawBody, seeded.environmentID, "masked 404 leaked the environment id: %s", res.RawBody)
			require.Len(t, ctrlClient.RetryVerificationCalls, callsBefore,
				"a rejected request must not restart anything")
		})
	}
}

// TestVerifyDomainExistenceNotLeaked asserts that a zero-permission root key targeting
// a real domain and a nonexistent one receives responses that are indistinguishable
// apart from the request id. Without this, permission rejections would act as an
// existence oracle over every domain in the workspace, and names are guessable. Do not
// weaken this by returning 403 for the real domain.
func TestVerifyDomainExistenceNotLeaked(t *testing.T) {
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
	require.Empty(t, ctrlClient.RetryVerificationCalls, "neither probe may reach ctrl")
}

// TestVerifyDomainExistenceNotLeakedWithPartialGrant covers the subtler oracle: a key
// that may verify one environment's domains must not learn about another environment's.
func TestVerifyDomainExistenceNotLeakedWithPartialGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	granted := seedDomain(t, h, nil)
	other := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(granted.workspaceID, fmt.Sprintf("environment.%s.verify_domain", granted.environmentID))
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
	require.Empty(t, ctrlClient.RetryVerificationCalls, "the ungranted domain must stay untouched")
}

// TestVerifyDomainVerifiedStateNotLeakedWithoutPermission pins the ordering of the
// permission check and the verified gate: a caller without the permission gets the
// masked 404, never the 412 that would reveal the domain exists and is verified.
func TestVerifyDomainVerifiedStateNotLeakedWithoutPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, verifiedDomain)
	rootKey := h.CreateRootKey(seeded.workspaceID)

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Empty(t, ctrlClient.RetryVerificationCalls)
}
