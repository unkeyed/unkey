package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_get_domain"
)

// TestGetDomainBadRequest covers the spec layer: the OpenAPI middleware rejects every
// case below against DomainIdentifier before the handler runs. The identifier accepts
// either an id (^[a-zA-Z0-9_]+$) or an FQDN, and nothing in between, so a malformed
// name fails here rather than becoming a lookup that can never match.
func TestGetDomainBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.read_domain")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name       string
		identifier string
	}{
		{name: "empty", identifier: ""},
		{name: "below min length", identifier: "ab"},
		{name: "leading dot", identifier: ".acme.com"},
		{name: "trailing dot", identifier: "api.acme.com."},
		{name: "consecutive dots", identifier: "api..acme.com"},
		{name: "label starts with hyphen", identifier: "-api.acme.com"},
		{name: "label ends with hyphen", identifier: "api-.acme.com"},
		{name: "single letter tld", identifier: "api.acme.c"},
		{name: "scheme included", identifier: "https://api.acme.com"},
		{name: "path included", identifier: "api.acme.com/v1"},
		{name: "port included", identifier: "api.acme.com:8080"},
		{name: "whitespace", identifier: "api acme.com"},
		{name: "leading whitespace", identifier: " api.acme.com"},
		{name: "wildcard", identifier: "*.acme.com"},
		{name: "traversal", identifier: "../api.acme.com"},
		{name: "id with a hyphen is neither form", identifier: "dom-1234abcd"},
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
}
