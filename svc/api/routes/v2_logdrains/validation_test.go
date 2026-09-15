package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
	"google.golang.org/protobuf/proto"
)

func TestLogdrainsRejectInvalidInput(t *testing.T) {
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
	update := &logdrains.Update{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(update)
	for _, tc := range []struct {
		name  string
		route zen.Route
		body  string
	}{
		{"update requires change", update, `{"logdrainId":"` + id + `"}`},
		{"failure status is internal", update, `{"logdrainId":"` + id + `","status":"paused_by_failure"}`},
		{"destination kind cannot change", update, `{"logdrainId":"` + id + `","name":"Changed","destination":{"axiom":{"dataset":"logs","token":"secret"}}}`},
		{"update cannot preserve unknown header", update, `{"logdrainId":"` + id + `","name":"Changed","destination":{"http":{"headers":[{"name":"Authorization","mode":"preserve"}]}}}`},
		{"update filter must match stream", update, `{"logdrainId":"` + id + `","name":"Changed","filters":{"passed":[true]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.CallRoute[json.RawMessage, openapi.BadRequestErrorResponse](h, tc.route, headers, json.RawMessage(tc.body))
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
