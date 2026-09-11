package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestCreateHTTPStreamFilters(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &logdrains.Get{DB: h.DB}
	h.Register(route)
	h.Register(get)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 10 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	for _, tc := range []struct{ stream, filters string }{
		{"audit_logs", `{"eventTypes":["key.create"]}`},
		{"ratelimits", `{"namespaceIds":["ns_one"],"passed":[false]}`},
		{"key_verifications", `{"outcomes":["VALID"],"keySpaceIds":["ks_one"]}`},
		{"gateway_requests", `{"statusClasses":[2,5],"projectIds":["proj_one"],"appIds":[],"environmentIds":[]}`},
		{"runtime_logs", `{"severities":["warn"],"projectIds":[],"appIds":["app_one"],"environmentIds":[]}`},
	} {
		t.Run(tc.stream, func(t *testing.T) {
			input := []byte(`{"name":"HTTP logs","stream":"` + tc.stream + `","filters":` + tc.filters + `,"destination":{"http":{"url":"https://logs.example.com","format":"ndjson","headers":[{"name":"Authorization","mode":"set","value":"secret-token"}]}}}`)
			response := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers, input)
			require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
			result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, get, headers, openapi.LogdrainIdRequest{LogdrainId: response.Body.Data.Id})
			require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
			require.NotContains(t, string(result.RawBody), "secret-token")
			encoded, err := json.Marshal(result.Body.Data.Filters)
			require.NoError(t, err)
			require.JSONEq(t, tc.filters, string(encoded))
		})
	}
}
