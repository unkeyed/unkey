package logdrains_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_get_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestGetCanonicalPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id, otherID := uid.New("ld"), uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"}}})
	require.NoError(t, err)
	for _, drainID := range []string{id, otherID} {
		_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Drain', 'audit_logs', ?, '', '', 123)", drainID, workspaceID, config)
		require.NoError(t, err)
	}
	for _, tc := range []struct {
		name, workspace, resource, action string
		status                            int
	}{
		{"specific read", workspaceID, id, "read", http.StatusOK},
		{"collection read", workspaceID, "*", "read", http.StatusOK},
		{"other drain", workspaceID, otherID, "read", http.StatusForbidden},
		{"foreign grant", h.CreateWorkspace().ID, "*", "read", http.StatusForbidden},
		{"write", workspaceID, id, "write", http.StatusForbidden},
		{"delete", workspaceID, "*", "delete", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := h.CreateRootKey(workspaceID, urn.New().Workspace(tc.workspace).Logdrain(tc.resource).String()+"#"+tc.action)
			result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.LogdrainIdRequest{LogdrainId: id})
			require.Equal(t, tc.status, result.Status, "%s", result.RawBody)
			if tc.status == http.StatusOK {
				require.Equal(t, id, result.Body.Data.Id)
			} else {
				require.NotContains(t, string(result.RawBody), "https://logs.example.com")
			}
		})
	}
}
