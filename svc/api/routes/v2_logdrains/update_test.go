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
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
	"google.golang.org/protobuf/proto"
)

func TestUpdateResumesFailedDrainWithoutResettingCursor(t *testing.T) {
	h := testutil.NewHarness(t)
	create := logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	update := &logdrains.Update{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &logdrains.Get{DB: h.DB}
	h.Register(&create)
	h.Register(update)
	h.Register(get)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	created := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, &create, headers, json.RawMessage(`{"name":"HTTP","stream":"ratelimits","filters":{"namespaceIds":["ns_keep"],"passed":[false]},"destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"set","value":"secret"}]}}}`))
	require.Equal(t, http.StatusOK, created.Status, "%s", created.RawBody)
	id := created.Body.Data.Id
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE logdrains SET status = 'paused_by_failure', consecutive_failures = 8, next_attempt_at = 9000, lease_expires_at = 9999, committed_offset_inserted_at = 3456, committed_offset_event_id = 'event_keep' WHERE id = ?", id)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 0 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	response := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, update, headers, json.RawMessage(`{"logdrainId":"`+id+`","batchSize":31,"filters":{"passed":[true]},"destination":{"http":{"headers":[{"name":"authorization","mode":"preserve"}]}}}`))
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, get, headers, openapi.LogdrainIdRequest{LogdrainId: id})
	require.Equal(t, http.StatusOK, result.Status)
	require.Equal(t, "running", string(result.Body.Data.Status))
	require.Equal(t, int64(31), result.Body.Data.BatchSize)
	require.Equal(t, []string{"ns_keep"}, *result.Body.Data.Filters.NamespaceIds)
	require.Equal(t, []bool{true}, *result.Body.Data.Filters.Passed)
	require.Equal(t, []string{"Authorization"}, result.Body.Data.Destination.Http.Headers)
	var lease, failures, next, offset int64
	var event string
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT lease_expires_at, consecutive_failures, next_attempt_at, committed_offset_inserted_at, committed_offset_event_id FROM logdrains WHERE id = ?", id).Scan(&lease, &failures, &next, &offset, &event))
	require.Equal(t, []int64{0, 0, 0, 3456}, []int64{lease, failures, next, offset})
	require.Equal(t, "event_keep", event)
}

func TestUpdateDistinguishesUserPauseFromFailurePause(t *testing.T) {
	h := testutil.NewHarness(t)
	update := &logdrains.Update{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &logdrains.Get{DB: h.DB}
	h.Register(update)
	h.Register(get)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"}}})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, initial, patch, status string
		failures                     int
	}{
		{"rename preserves failure pause", "paused_by_failure", `"name":"Renamed"`, "paused_by_failure", 8},
		{"delivery changes preserve user pause", "paused_by_user", `"batchSize":23`, "paused_by_user", 0},
		{"explicit resume clears failures", "paused_by_user", `"status":"running"`, "running", 0},
		{"explicit pause", "running", `"status":"paused_by_user"`, "paused_by_user", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uid.New("ld")
			_, err := h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, status, consecutive_failures, committed_offset_inserted_at, lease_id, fencing_token, created_at) VALUES (?, ?, 'Pause', 'audit_logs', ?, ?, 8, 4321, '', '', 123)", id, workspaceID, config, tc.initial)
			require.NoError(t, err)
			result := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, update, headers, json.RawMessage(`{"logdrainId":"`+id+`",`+tc.patch+`}`))
			require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
			read := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, get, headers, openapi.LogdrainIdRequest{LogdrainId: id})
			require.Equal(t, http.StatusOK, read.Status)
			require.Equal(t, tc.status, string(read.Body.Data.Status))
			require.Equal(t, tc.failures, read.Body.Data.ConsecutiveFailures)
			require.Equal(t, int64(4321), read.Body.Data.CommittedOffsetInsertedAt)
		})
	}
}
