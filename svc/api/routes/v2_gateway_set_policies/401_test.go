package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_set_policies"
)

func TestSetPoliciesUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer invalid_token"},
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:     "payments",
		App:         "payments-api",
		Environment: "env_1234abcd",
		Policies:    []openapi.Policy{firewallPolicy("deny", true)},
	})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
