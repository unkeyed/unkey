package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestLogdrainsRequireAuthenticationAndPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	foreignID := uid.New("ld")
	_, err := h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Foreign', 'audit_logs', '', '', '', 123)", foreignID, h.CreateWorkspace().ID)
	require.NoError(t, err)
	denied := h.CreateRootKey(workspaceID, "api.*.read_api")
	allowed := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	for _, tc := range []struct {
		route   zen.Route
		body    string
		foreign bool
	}{
		{&logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}, `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`, false},
		{&logdrains.List{DB: h.DB}, `{}`, false},
		{&logdrains.Get{DB: h.DB}, `{"logdrainId":"` + foreignID + `"}`, true},
		{&logdrains.Update{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}, `{"logdrainId":"` + foreignID + `","status":"running"}`, true},
		{&logdrains.Delete{DB: h.DB, Auditlogs: h.Auditlogs}, `{"logdrainId":"` + foreignID + `"}`, true},
		{&logdrains.Metrics{DB: h.DB, ClickHouse: h.ClickHouse, Clock: h.Clock}, `{"logdrainId":"` + foreignID + `","hours":24}`, true},
		{&logdrains.Deliveries{DB: h.DB, ClickHouse: h.ClickHouse, Clock: h.Clock}, `{"logdrainId":"` + foreignID + `"}`, true},
	} {
		t.Run(tc.route.Path(), func(t *testing.T) {
			h.Register(tc.route)
			for _, auth := range []struct {
				key    string
				status int
			}{{"invalid", http.StatusUnauthorized}, {denied, http.StatusForbidden}} {
				result := testutil.CallRoute[json.RawMessage, json.RawMessage](h, tc.route, http.Header{"Authorization": {"Bearer " + auth.key}, "Content-Type": {"application/json"}}, json.RawMessage(tc.body))
				require.Equal(t, auth.status, result.Status, "%s", result.RawBody)
			}
			if tc.foreign {
				result := testutil.CallRoute[json.RawMessage, json.RawMessage](h, tc.route, http.Header{"Authorization": {"Bearer " + allowed}, "Content-Type": {"application/json"}}, json.RawMessage(tc.body))
				require.Equal(t, http.StatusNotFound, result.Status, "%s", result.RawBody)
			}
		})
	}
	var name string
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT name FROM logdrains WHERE id = ?", foreignID).Scan(&name))
	require.Equal(t, "Foreign", name)
}
