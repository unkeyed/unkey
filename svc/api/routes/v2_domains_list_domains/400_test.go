package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

// TestListDomainsBadRequest covers the spec layer: the OpenAPI middleware rejects
// every case below before the handler runs. project/app/environment are
// ResourceIdentifier (minLength 3, maxLength 255, ^[a-zA-Z0-9_-]+$) and search is
// capped at 256 characters.
func TestListDomainsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")
	headers := authHeaders(rootKey)

	withEnv := func(mutate func(*handler.Request)) handler.Request {
		req := makeRequest(env)
		mutate(&req)
		return req
	}

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "empty project", req: withEnv(func(r *handler.Request) { r.Project = "" })},
		{name: "empty app", req: withEnv(func(r *handler.Request) { r.App = "" })},
		{name: "empty environment", req: withEnv(func(r *handler.Request) { r.Environment = "" })},
		{name: "project with illegal character", req: withEnv(func(r *handler.Request) { r.Project = "pay ments" })},
		{name: "environment with a dot", req: withEnv(func(r *handler.Request) { r.Environment = "prod.uction" })},
		{name: "search over 256 chars", req: withEnv(func(r *handler.Request) { r.Search = ptr.P(strings.Repeat("a", 257)) })},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
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
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, tc.headers, makeRequest(env))
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
