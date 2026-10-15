package testutil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Bearer returns JSON request headers with the supplied credential.
func Bearer(credential string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + credential},
	}
}

// Authentication checks a registered route with an unknown root key before
// exercising the same request with valid credentials. It does not register
// middleware or replace credential verification.
type Authentication[Req, Res any] struct {
	Request       Req
	ValidHeaders  http.Header
	Status        int
	VerifyAllowed func(*testing.T, TestResponse[Res])
	VerifyDenied  func(*testing.T)
}

// CheckAuthentication requires rejection without business effects and a
// successful positive control.
func CheckAuthentication[Req, Res any](t *testing.T, h *Harness, route zen.Route, contract Authentication[Req, Res]) {
	t.Helper()
	require.NotNil(t, contract.VerifyAllowed)
	require.NotNil(t, contract.VerifyDenied)
	require.NotEmpty(t, contract.ValidHeaders)
	require.GreaterOrEqual(t, contract.Status, 200)
	require.Less(t, contract.Status, 300)

	if !t.Run("unknown credential", func(t *testing.T) {
		requestHarness := *h
		requestHarness.t = t

		res := CallRoute[Req, openapi.UnauthorizedErrorResponse](
			&requestHarness,
			route,
			Bearer(uid.New(uid.KeyPrefix)),
			contract.Request,
		)
		require.Equal(t, http.StatusUnauthorized, res.Status, "%s", res.RawBody)
		require.Equal(t, openapi.BaseError{
			Status: http.StatusUnauthorized,
			Title:  "Unauthorized",
			Type:   "https://unkey.com/docs/errors/unkey/authentication/key_not_found",
			Detail: "The provided root key is invalid. We could not find the requested key.",
		}, res.Body.Error)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		contract.VerifyDenied(t)
	}) {
		return
	}

	t.Run("valid credential", func(t *testing.T) {
		requestHarness := *h
		requestHarness.t = t

		res := CallRoute[Req, Res](&requestHarness, route, contract.ValidHeaders, contract.Request)
		require.Equal(t, contract.Status, res.Status, "%s", res.RawBody)

		contract.VerifyAllowed(t, res)
	})
}

// Grant describes real root-key credentials. An empty Permissions slice grants
// nothing; it never defaults to administrator access.
type Grant struct {
	WorkspaceID string
	Permissions []string
}

// AllowedRequest names a request and an independently specified sufficient grant.
type AllowedRequest[Req any] struct {
	Name    string
	Grant   Grant
	Request Req
	Status  int
}

// DeniedRequest pins the complete public error for an insufficient grant.
type DeniedRequest[Req any] struct {
	Name    string
	Grant   Grant
	Request Req
	Error   openapi.BaseError
}

// Authorization runs denials before positive controls. Callers own request
// fixtures, expected grants, business-state checks, and independent state for
// non-repeatable allowed operations such as deletion.
type Authorization[Req, Res any] struct {
	Denied        []DeniedRequest[Req]
	Allowed       []AllowedRequest[Req]
	VerifyDenied  func(*testing.T, Req)
	VerifyAllowed func(*testing.T, Req, TestResponse[Res])
}

// CheckAuthorization authenticates real root keys without bypassing the route's
// middleware. Expected permissions are supplied by the test, never the handler.
func CheckAuthorization[Req, Res any](t *testing.T, h *Harness, route zen.Route, contract Authorization[Req, Res]) {
	t.Helper()
	require.NotEmpty(t, contract.Denied)
	require.NotEmpty(t, contract.Allowed)
	require.NotNil(t, contract.VerifyDenied)
	require.NotNil(t, contract.VerifyAllowed)

	deniedPassed := true
	for _, tc := range contract.Denied {
		rootKey := h.CreateRootKey(tc.Grant.WorkspaceID, tc.Grant.Permissions...)
		passed := t.Run(tc.Name, func(t *testing.T) {
			requestHarness := *h
			requestHarness.t = t
			require.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, tc.Error.Status)

			res := CallRoute[Req, openapi.ForbiddenErrorResponse](
				&requestHarness,
				route,
				Bearer(rootKey),
				tc.Request,
			)
			require.Equal(t, tc.Error.Status, res.Status, "%s", res.RawBody)
			require.Equal(t, tc.Error, res.Body.Error)
			require.NotEmpty(t, res.Body.Meta.RequestId)

			contract.VerifyDenied(t, tc.Request)
		})
		deniedPassed = deniedPassed && passed
	}
	if !deniedPassed {
		return
	}

	for _, tc := range contract.Allowed {
		rootKey := h.CreateRootKey(tc.Grant.WorkspaceID, tc.Grant.Permissions...)
		t.Run(tc.Name, func(t *testing.T) {
			requestHarness := *h
			requestHarness.t = t
			require.GreaterOrEqual(t, tc.Status, 200)
			require.Less(t, tc.Status, 300)

			res := CallRoute[Req, Res](&requestHarness, route, Bearer(rootKey), tc.Request)
			require.Equal(t, tc.Status, res.Status, "%s", res.RawBody)

			contract.VerifyAllowed(t, tc.Request, res)
		})
	}
}
