package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
)

func TestCreateRequiresAuthenticationAndPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	denied := h.CreateRootKey(workspaceID, "api.*.read_api")
	foreign := h.CreateRootKey(workspaceID, "unkey:v1:"+h.CreateWorkspace().ID+":logdrains/*#write")
	revoked := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `keys` SET deleted_at_m = ? WHERE hash = ?", h.Clock.Now().UnixMilli(), hash.Sha256(revoked))
	require.NoError(t, err)
	for _, tc := range []struct {
		name, authorization string
		status              int
	}{
		{"missing authorization", "", http.StatusBadRequest},
		{"malformed authorization", "Basic invalid", http.StatusBadRequest},
		{"missing bearer token", "Bearer", http.StatusBadRequest},
		{"invalid key", "Bearer invalid", http.StatusUnauthorized},
		{"revoked key", "Bearer " + revoked, http.StatusUnauthorized},
		{"insufficient permission", "Bearer " + denied, http.StatusForbidden},
		{"permission for another workspace", "Bearer " + foreign, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			headers := http.Header{"Content-Type": {"application/json"}}
			if tc.authorization != "" {
				headers.Set("Authorization", tc.authorization)
			}
			result := testutil.CallRoute[json.RawMessage, openapi.UnauthorizedErrorResponse](h, route, headers, json.RawMessage(`{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`))
			require.Equal(t, tc.status, result.Status, "%s", result.RawBody)
			require.Equal(t, tc.status, result.Body.Error.Status)
			require.NotEmpty(t, result.Body.Error.Type)
			require.NotEmpty(t, result.Body.Error.Detail)
			require.NotEmpty(t, result.Body.Meta.RequestId)
			var count int
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&count))
			require.Zero(t, count)
		})
	}
}
