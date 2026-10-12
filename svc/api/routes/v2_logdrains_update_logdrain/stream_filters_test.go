package logdrains_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

func TestUpdateClearsSelectedFilterAndPreservesSiblingDimensions(t *testing.T) {
	for _, stream := range []string{"audit_logs", "key_verifications", "gateway_requests", "runtime_logs"} {
		t.Run(stream, func(t *testing.T) {
			h, route, id, expected := seedUpdateDrain(t)
			var patch string
			switch stream {
			case "audit_logs":
				expected.Stream = &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{EventTypes: []string{"key.create"}}}
				patch = `"eventTypes":[]`
			case "key_verifications":
				expected.Stream = &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{
					Outcomes:    []string{"VALID"},
					KeySpaceIds: []string{"ks_keep"},
				}}
				patch = `"outcomes":[]`
			case "gateway_requests":
				expected.Stream = &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{
					StatusClasses:  []logdrainv1.HttpStatusClass{logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_5XX},
					ProjectIds:     []string{"proj_keep"},
					AppIds:         []string{"app_keep"},
					EnvironmentIds: []string{"env_keep"},
				}}
				patch = `"statusClasses":[]`
			case "runtime_logs":
				expected.Stream = &logdrainv1.Config_RuntimeLogs{RuntimeLogs: &logdrainv1.RuntimeLogStreamConfig{
					Severities:     []string{"error"},
					ProjectIds:     []string{"proj_keep"},
					AppIds:         []string{"app_keep"},
					EnvironmentIds: []string{"env_keep"},
				}}
				patch = `"severities":[]`
			}
			encoded, err := proto.Marshal(expected)
			require.NoError(t, err)
			_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE logdrains SET stream = ?, config = ? WHERE id = ?", stream, encoded, id)
			require.NoError(t, err)
			key := h.CreateRootKey(h.Resources().UserWorkspace.ID, urn.New().Workspace(h.Resources().UserWorkspace.ID).Logdrain(id).String()+"#write")
			res := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, updateHeaders(key), json.RawMessage(`{"logdrainId":"`+id+`","filters":{`+patch+`}}`))
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			switch stream {
			case "audit_logs":
				expected.GetAuditLogs().EventTypes = nil
			case "key_verifications":
				expected.GetKeyVerifications().Outcomes = nil
			case "gateway_requests":
				expected.GetGatewayRequests().StatusClasses = nil
			case "runtime_logs":
				expected.GetRuntimeLogs().Severities = nil
			}
			assertUpdateState(t, h, id, "Original", expected, true)
		})
	}
}
