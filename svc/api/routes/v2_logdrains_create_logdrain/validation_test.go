package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestCreateRejectsInvalidInputWithoutChangingExistingDrains(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"}}})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Unchanged', 'audit_logs', ?, '', '', 123)", id, workspaceID, config)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 2 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	for _, tc := range []struct{ name, body string }{
		{"missing create fields", `{}`},
		{"empty name", `{"name":" ","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"zero batch size", `{"name":"Logs","stream":"audit_logs","batchSize":0,"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"mismatched filter", `{"name":"Logs","stream":"audit_logs","filters":{"passed":[false]},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"empty filter value", `{"name":"Logs","stream":"audit_logs","filters":{"eventTypes":[" "]},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"two destinations", `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"},"axiom":{"dataset":"logs","token":"secret"}}}`},
		{"URL credentials", `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://user:secret@logs.example.com"}}}`},
		{"duplicate header names", `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"set","value":"secret"},{"name":"authorization","mode":"set","value":"other"}]}}}`},
		{"preserve on create", `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"preserve"}]}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.CallRoute[json.RawMessage, openapi.BadRequestErrorResponse](h, route, headers, json.RawMessage(tc.body))
			require.Equal(t, http.StatusBadRequest, result.Status, "%s", result.RawBody)
			require.Equal(t, http.StatusBadRequest, result.Body.Error.Status)
			require.Contains(t, result.Body.Error.Type, "application/invalid_input")
			require.NotEmpty(t, result.Body.Error.Detail)
			require.NotEmpty(t, result.Body.Meta.RequestId)
			var name, status string
			var stored []byte
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT name, status, config FROM logdrains WHERE id = ?", id).Scan(&name, &status, &stored))
			require.Equal(t, "Unchanged", name)
			require.Equal(t, "running", status)
			require.Equal(t, config, stored)
			require.Empty(t, h.FindAuditLogsByTargetID(context.Background(), t, id))
			var count int
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}
