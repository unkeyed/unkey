package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestLogdrainsRequireAuthenticationAndPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	foreignID := uid.New("ld")
	_, err := h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Foreign', 'audit_logs', '', '', '', 123)", foreignID, h.CreateWorkspace().ID)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	denied := h.CreateRootKey(workspaceID, "api.*.read_api")
	allowed := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	revoked := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `keys` SET deleted_at_m = ? WHERE hash = ?", h.Clock.Now().UnixMilli(), hash.Sha256(revoked))
	require.NoError(t, err)
	for _, tc := range []struct {
		route   zen.Route
		body    string
		foreign bool
	}{
		{&logdrains.Get{DB: h.DB}, `{"logdrainId":"` + foreignID + `"}`, true},
		{&logdrains.List{DB: h.DB}, `{}`, false},
	} {
		t.Run(tc.route.Path(), func(t *testing.T) {
			h.Register(tc.route)
			for _, auth := range []struct {
				name, authorization string
				status              int
			}{
				{"missing authorization", "", http.StatusBadRequest},
				{"malformed authorization", "Basic invalid", http.StatusBadRequest},
				{"missing bearer token", "Bearer", http.StatusBadRequest},
				{"invalid key", "Bearer invalid", http.StatusUnauthorized},
				{"revoked key", "Bearer " + revoked, http.StatusUnauthorized},
				{"insufficient permission", "Bearer " + denied, http.StatusForbidden},
			} {
				t.Run(auth.name, func(t *testing.T) {
					headers := http.Header{"Content-Type": {"application/json"}}
					if auth.authorization != "" {
						headers.Set("Authorization", auth.authorization)
					}
					result := testutil.CallRoute[json.RawMessage, openapi.UnauthorizedErrorResponse](h, tc.route, headers, json.RawMessage(tc.body))
					require.Equal(t, auth.status, result.Status, "%s", result.RawBody)
					require.Equal(t, auth.status, result.Body.Error.Status)
					require.NotEmpty(t, result.Body.Error.Type)
					require.NotEmpty(t, result.Body.Error.Detail)
					require.NotEmpty(t, result.Body.Meta.RequestId)
				})
			}
			if tc.foreign {
				// Foreign IDs return 404 so callers cannot discover another workspace's drains.
				result := testutil.CallRoute[json.RawMessage, openapi.NotFoundErrorResponse](h, tc.route, http.Header{"Authorization": {"Bearer " + allowed}, "Content-Type": {"application/json"}}, json.RawMessage(tc.body))
				require.Equal(t, http.StatusNotFound, result.Status, "%s", result.RawBody)
				require.Equal(t, http.StatusNotFound, result.Body.Error.Status)
				require.Equal(t, "https://unkey.com/docs/errors/unkey/data/logdrain_not_found", result.Body.Error.Type)
				require.NotEmpty(t, result.Body.Meta.RequestId)
			}
		})
	}
	var name string
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT name FROM logdrains WHERE id = ?", foreignID).Scan(&name))
	require.Equal(t, "Foreign", name)
	require.Empty(t, h.FindAuditLogsByTargetID(context.Background(), t, foreignID))
	var count int
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&count))
	require.Zero(t, count)
}
