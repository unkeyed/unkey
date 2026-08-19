package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
)

func TestUpdatePolicyUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer invalid_token"},
	}
	req := makeRequest(seededEnv{
		workspaceID:   "",
		projectID:     uid.New(uid.ProjectPrefix),
		appID:         uid.New(uid.AppPrefix),
		environmentID: uid.New(uid.EnvironmentPrefix),
	}, uid.New(uid.PolicyPrefix))
	req.Name = ptr.P("KEBAP")
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
