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
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_delete_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestDeleteCanonicalPermissionsAndRetry(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"}}})
	require.NoError(t, err)
	otherID := uid.New("ld")
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Other', 'audit_logs', ?, '', '', 123)", otherID, workspaceID, config)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, workspace, resource, action string
		status                            int
	}{
		{"specific delete", workspaceID, "", "delete", http.StatusOK},
		{"collection delete", workspaceID, "*", "delete", http.StatusOK},
		{"other drain", workspaceID, otherID, "delete", http.StatusForbidden},
		{"foreign grant", h.CreateWorkspace().ID, "*", "delete", http.StatusForbidden},
		{"read", workspaceID, "*", "read", http.StatusForbidden},
		{"write", workspaceID, "", "write", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uid.New("ld")
			_, err := h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Unchanged', 'audit_logs', ?, '', '', 123)", id, workspaceID, config)
			require.NoError(t, err)
			resource := tc.resource
			if resource == "" {
				resource = id
			}
			key := h.CreateRootKey(workspaceID, urn.New().Workspace(tc.workspace).Logdrain(resource).String()+"#"+tc.action)
			headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
			input := openapi.LogdrainIdRequest{LogdrainId: id}
			result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainMutationResponse](h, route, headers, input)
			require.Equal(t, tc.status, result.Status, "%s", result.RawBody)
			if tc.status == http.StatusOK {
				require.Equal(t, id, result.Body.Data.Id)
				retry := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.NotFoundErrorResponse](h, route, headers, input)
				require.Equal(t, http.StatusNotFound, retry.Status, "%s", retry.RawBody)
				require.Equal(t, "https://unkey.com/docs/errors/unkey/data/logdrain_not_found", retry.Body.Error.Type)
				require.NotEmpty(t, retry.Body.Meta.RequestId)
				events := h.FindAuditLogsByTargetID(t.Context(), t, id)
				require.Len(t, events, 1)
				require.Equal(t, "logdrain.delete", events[0].Event)
				var count int
				require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM logdrains WHERE id = ?", id).Scan(&count))
				require.Zero(t, count)
			} else {
				var name, status string
				var stored []byte
				require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT name, status, config FROM logdrains WHERE id = ?", id).Scan(&name, &status, &stored))
				require.Equal(t, "Unchanged", name)
				require.Equal(t, "running", status)
				require.Equal(t, config, stored)
				require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, id))
			}
		})
	}
	key := h.CreateRootKey(workspaceID, urn.New().Workspace(workspaceID).Logdrain("*").String()+"#delete")
	id := uid.New("ld")
	missing := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.NotFoundErrorResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.LogdrainIdRequest{LogdrainId: id})
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/logdrain_not_found", missing.Body.Error.Type)
	require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, id))
	var name string
	require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT name FROM logdrains WHERE id = ?", otherID).Scan(&name))
	require.Equal(t, "Other", name)
}
