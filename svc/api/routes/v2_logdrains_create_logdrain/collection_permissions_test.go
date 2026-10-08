package logdrains_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestCreateRequiresCollectionWriteWithoutSideEffects(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{
		DB:          h.DB,
		Vault:       h.Vault,
		Auditlogs:   h.Auditlogs,
		Clock:       h.Clock,
		LimitsCache: h.Caches.WorkspaceLimits,
	}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"}}})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Unchanged', 'audit_logs', ?, '', '', 123)", id, workspaceID, config)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE `limits` SET logdrains_max = 2 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	var auditsBefore int
	require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM clickhouse_outbox WHERE workspace_id = ?", workspaceID).Scan(&auditsBefore))
	for _, tc := range []struct{ name, resource, action string }{
		{"concrete write", id, "write"},
		{"collection read", "*", "read"},
		{"collection delete", "*", "delete"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := h.CreateRootKey(workspaceID, urn.New().Workspace(workspaceID).Logdrain(tc.resource).String()+"#"+tc.action)
			result := testutil.CallRoute[json.RawMessage, openapi.ForbiddenErrorResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, json.RawMessage(`{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com"}}}`))
			require.Equal(t, http.StatusForbidden, result.Status, "%s", result.RawBody)
			require.Equal(t, http.StatusForbidden, result.Body.Error.Status)
			require.Contains(t, result.Body.Error.Type, "authorization/insufficient_permissions")
			require.NotEmpty(t, result.Body.Meta.RequestId)
			var count int
			require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&count))
			require.Equal(t, 1, count)
			var name string
			var stored []byte
			require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT name, config FROM logdrains WHERE id = ?", id).Scan(&name, &stored))
			require.Equal(t, "Unchanged", name)
			require.Equal(t, config, stored)
			require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, id))
			var auditsAfter int
			require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM clickhouse_outbox WHERE workspace_id = ?", workspaceID).Scan(&auditsAfter))
			require.Equal(t, auditsBefore, auditsAfter)
		})
	}
}
