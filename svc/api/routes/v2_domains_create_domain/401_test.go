package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

// TestCreateDomainUnauthorized covers a well-formed bearer token that does not
// resolve to a key. A header that is absent or missing the 'Bearer ' prefix never
// reaches key lookup and is reported as a 400 instead, so those live in
// 400_test.go.
func TestCreateDomainUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, CtrlClient: &testutil.MockCustomDomainClient{}, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)

	env := seedEnvironment(t, h)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, authHeaders("invalid_token"), makeRequest(env, randomDomain()))
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
