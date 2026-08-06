package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_verify_domain"
)

// TestVerifyDomainBadRequest covers the spec layer: the identifier carries only length
// bounds, since neither a Punycode-aware name rule nor the id shape is expressible as
// one pattern. Everything shape-related is the handler's job — an identifier that is
// neither a parseable name nor a stored id misses the lookup and 404s, covered in
// TestVerifyDomainNotFound.
func TestVerifyDomainBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name       string
		identifier string
	}{
		{name: "empty", identifier: ""},
		{name: "below min length", identifier: "ab"},
		{name: "over 253 chars", identifier: strings.Repeat("a", 250) + ".com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
				Domain: tc.identifier,
			})
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	// A header that never reaches key lookup is a 400, not a 401: an absent header
	// fails the security scheme, and one without the 'Bearer ' prefix fails parsing.
	authCases := []struct {
		name    string
		headers http.Header
	}{
		{name: "missing authorization header", headers: http.Header{"Content-Type": {"application/json"}}},
		{
			name: "authorization header without bearer prefix",
			headers: http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"unkey_1234abcd"},
			},
		},
	}

	for _, tc := range authCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, tc.headers, handler.Request{
				Domain: seeded.domain,
			})
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	require.Empty(t, ctrlClient.RetryVerificationCalls,
		"a request rejected by the schema must never reach ctrl")
}
