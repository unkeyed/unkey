package logdrains_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_list_logdrains"
	"google.golang.org/protobuf/proto"
)

func TestListEmptyWorkspaceSerializesArray(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, urn.New().Workspace(workspaceID).Logdrain("*").String()+"#read")
	result := testutil.CallRoute[openapi.ListLogdrainsRequest, openapi.ListLogdrainsResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.ListLogdrainsRequest{})
	require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
	require.Contains(t, string(result.RawBody), `"data":[]`)
	require.Empty(t, result.Body.Data)
	require.False(t, result.Body.Pagination.HasMore)
	require.Nil(t, result.Body.Pagination.Cursor)
	require.NotContains(t, string(result.RawBody), `"cursor"`)
	require.NotEmpty(t, result.Body.Meta.RequestId)
}

func TestListDefaultLimitAndInclusiveCursors(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	prefix := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Axiom{Axiom: &logdrainv1.AxiomConfig{
		Dataset:        "logs",
		EncryptedToken: "secret-ciphertext",
	}}})
	require.NoError(t, err)
	for i := range 101 {
		_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Drain', 'audit_logs', ?, '', '', 123)", fmt.Sprintf("%s_%03d", prefix, i), workspaceID, config)
		require.NoError(t, err)
	}
	foreignID := prefix + "_050a"
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Foreign secret', 'audit_logs', ?, '', '', 123)", foreignID, h.CreateWorkspace().ID, config)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, urn.New().Workspace(workspaceID).Logdrain("*").String()+"#read")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	first := testutil.CallRoute[openapi.ListLogdrainsRequest, openapi.ListLogdrainsResponse](h, route, headers, openapi.ListLogdrainsRequest{})
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Len(t, first.Body.Data, 100)
	for i, drain := range first.Body.Data {
		require.Equal(t, fmt.Sprintf("%s_%03d", prefix, i), drain.Id)
	}
	require.True(t, first.Body.Pagination.HasMore)
	require.Equal(t, new(prefix+"_100"), first.Body.Pagination.Cursor)
	for _, tc := range []struct {
		name, cursor, firstID string
		count                 int
	}{
		{"returned cursor is inclusive", prefix + "_100", prefix + "_100", 1},
		{"first ID is inclusive", prefix + "_000", prefix + "_000", 100},
		{"nonexistent cursor between rows", prefix + "_050b", prefix + "_051", 50},
		{"foreign cursor cannot leak", foreignID, prefix + "_051", 50},
		{"cursor after last row", prefix + "_999", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.CallRoute[openapi.ListLogdrainsRequest, openapi.ListLogdrainsResponse](h, route, headers, openapi.ListLogdrainsRequest{Cursor: new(tc.cursor)})
			require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
			require.Len(t, result.Body.Data, tc.count)
			if tc.count > 0 {
				require.Equal(t, tc.firstID, result.Body.Data[0].Id)
			}
			require.Equal(t, tc.count == 100, result.Body.Pagination.HasMore)
			if tc.count < 100 {
				require.Nil(t, result.Body.Pagination.Cursor)
			}
			require.NotContains(t, string(result.RawBody), foreignID)
			require.NotContains(t, string(result.RawBody), "Foreign secret")
			require.NotContains(t, string(result.RawBody), "secret-ciphertext")
		})
	}
}

func TestListPopulatedAxiomPublicResponse(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{
		BatchSize: 42,
		Destination: &logdrainv1.Config_Axiom{Axiom: &logdrainv1.AxiomConfig{
			Dataset:        "production",
			EncryptedToken: "must-not-leak",
		}},
		Stream: &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{EventTypes: []string{"key.create", "key.delete"}}},
	})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, status, config, lease_id, fencing_token, consecutive_failures, committed_offset_inserted_at, created_at) VALUES (?, ?, 'Production logs', 'audit_logs', 'paused_by_user', ?, 'private-lease', 'private-fence', 7, 456, 123)", id, workspaceID, config)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, urn.New().Workspace(workspaceID).Logdrain("*").String()+"#read")
	result := testutil.CallRoute[openapi.ListLogdrainsRequest, openapi.ListLogdrainsResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.ListLogdrainsRequest{Limit: new(1)})
	require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
	var body struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.RawBody), &body))
	require.Len(t, body.Data, 1)
	require.JSONEq(t, fmt.Sprintf(`{"id":%q,"name":"Production logs","stream":"audit_logs","status":"paused_by_user","batchSize":42,"createdAt":123,"filters":{"eventTypes":["key.create","key.delete"]},"destination":{"axiom":{"dataset":"production"}}}`, id), string(body.Data[0]))
	require.False(t, result.Body.Pagination.HasMore)
	require.Nil(t, result.Body.Pagination.Cursor)
	require.NotEmpty(t, result.Body.Meta.RequestId)
}
