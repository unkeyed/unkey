package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

// TestCreateDomainBadRequest covers the spec layer only: the OpenAPI middleware
// rejects every case below against V2DomainsCreateDomainRequestBody before the
// handler runs. `domain` carries minLength 4, maxLength 253, and the FQDN
// pattern; project/app/environment are ResourceIdentifier (minLength 3,
// maxLength 255, ^[a-zA-Z0-9_-]+$). Handler-layer 400s come from ctrl and are
// covered by TestCreateDomainCtrlRejectsDomain.
func TestCreateDomainBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, CtrlClient: &testutil.MockCustomDomainClient{}}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "empty domain", req: makeRequest(env, "")},
		{name: "no dot", req: makeRequest(env, "localhost")},
		{name: "leading dot", req: makeRequest(env, ".acme.com")},
		{name: "trailing dot", req: makeRequest(env, "api.acme.com.")},
		{name: "label starts with hyphen", req: makeRequest(env, "-api.acme.com")},
		{name: "label ends with hyphen", req: makeRequest(env, "api-.acme.com")},
		{name: "underscore in label", req: makeRequest(env, "api_v2.acme.com")},
		{name: "single letter tld", req: makeRequest(env, "api.acme.c")},
		{name: "scheme included", req: makeRequest(env, "https://api.acme.com")},
		{name: "path included", req: makeRequest(env, "api.acme.com/v1")},
		{name: "port included", req: makeRequest(env, "api.acme.com:8080")},
		{name: "whitespace", req: makeRequest(env, "api acme.com")},
		{name: "wildcard", req: makeRequest(env, "*.acme.com")},
		{name: "over 253 chars", req: makeRequest(env, strings.Repeat("a", 250)+".com")},
		{name: "empty project", req: handler.Request{Project: "", App: env.appID, Environment: env.environmentID, Domain: "api.acme.com"}},
		{name: "empty app", req: handler.Request{Project: env.projectID, App: "", Environment: env.environmentID, Domain: "api.acme.com"}},
		{name: "empty environment", req: handler.Request{Project: env.projectID, App: env.appID, Environment: "", Domain: "api.acme.com"}},
		{name: "project with illegal character", req: handler.Request{Project: "pay ments", App: env.appID, Environment: env.environmentID, Domain: "api.acme.com"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	// A header that never reaches key lookup is a 400, not a 401: an absent header
	// fails the security scheme, and one without the 'Bearer ' prefix fails
	// parsing.
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
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, tc.headers, makeRequest(env, "api.acme.com"))
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}

// TestCreateDomainCtrlRejectsDomain covers the handler layer: ctrl runs its own
// domain validation and any InvalidArgument it returns must surface as a 400,
// not a 500.
func TestCreateDomainCtrlRejectsDomain(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid domain format"))
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
}
