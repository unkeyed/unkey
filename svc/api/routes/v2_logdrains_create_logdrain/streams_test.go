package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestCreateHTTPStreamFilters(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 10 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	for _, tc := range []struct {
		stream, filters string
		expected        *logdrainv1.Config
	}{
		{"audit_logs", `{"eventTypes":["key.create"]}`, &logdrainv1.Config{Stream: &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{EventTypes: []string{"key.create"}}}}},
		{"ratelimits", `{"namespaceIds":["ns_one"],"passed":[false]}`, &logdrainv1.Config{Stream: &logdrainv1.Config_Ratelimits{Ratelimits: &logdrainv1.RatelimitStreamConfig{NamespaceIds: []string{"ns_one"}, Passed: []bool{false}}}}},
		{"key_verifications", `{"outcomes":["VALID"],"keySpaceIds":["ks_one"]}`, &logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{Outcomes: []string{"VALID"}, KeySpaceIds: []string{"ks_one"}}}}},
		{"gateway_requests", `{"statusClasses":[2,5],"projectIds":["proj_one"],"appIds":[],"environmentIds":[]}`, &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{StatusClasses: []logdrainv1.HttpStatusClass{2, 5}, ProjectIds: []string{"proj_one"}}}}},
		{"runtime_logs", `{"severities":["warn"],"projectIds":[],"appIds":["app_one"],"environmentIds":[]}`, &logdrainv1.Config{Stream: &logdrainv1.Config_RuntimeLogs{RuntimeLogs: &logdrainv1.RuntimeLogStreamConfig{Severities: []string{"warn"}, AppIds: []string{"app_one"}}}}},
	} {
		t.Run(tc.stream, func(t *testing.T) {
			input := []byte(`{"name":"HTTP logs","stream":"` + tc.stream + `","filters":` + tc.filters + `,"destination":{"http":{"url":"https://logs.example.com","format":"ndjson","headers":[{"name":"Authorization","mode":"set","value":"secret-token"}]}}}`)
			response := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers, input)
			require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
			require.NotContains(t, string(response.RawBody), "secret-token")
			var stored []byte
			var stream string
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT stream, config FROM logdrains WHERE workspace_id = ? AND id = ?", workspaceID, response.Body.Data.Id).Scan(&stream, &stored))
			require.Equal(t, tc.stream, stream)
			require.NotContains(t, string(stored), "secret-token")
			config := &logdrainv1.Config{}
			require.NoError(t, proto.Unmarshal(stored, config))
			require.True(t, proto.Equal(tc.expected, &logdrainv1.Config{Stream: config.Stream}), "unexpected stream: %v", config.Stream)
			require.Equal(t, "https://logs.example.com", config.GetHttp().GetUrl())
			require.Equal(t, logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON, config.GetHttp().GetFormat())
			require.Len(t, config.GetHttp().GetHeaders(), 1)
			header := config.GetHttp().GetHeaders()[0]
			require.Equal(t, "Authorization", header.GetName())
			decrypted, err := h.Vault.Decrypt(context.Background(), &vaultv1.DecryptRequest{Keyring: workspaceID, Encrypted: header.GetEncryptedValue()})
			require.NoError(t, err)
			require.Equal(t, "secret-token", decrypted.GetPlaintext())
		})
	}
}
