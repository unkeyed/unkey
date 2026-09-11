package logdrains_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestRecentDeliveriesAreBoundedAndNewestFirst(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &logdrains.Deliveries{DB: h.DB, ClickHouse: h.ClickHouse, Clock: h.Clock}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	_, err := h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'History', 'audit_logs', '', '', '', 123)", id, workspaceID)
	require.NoError(t, err)
	now := h.Clock.Now().UnixMilli()
	for i := range 23 {
		timeMs := now - int64(i)
		if i == 22 {
			timeMs = now - 86_400_001
		}
		require.NoError(t, h.ClickHouse.Exec(context.Background(), "INSERT INTO default.logdrain_deliveries_raw_v1 (workspace_id, drain_id, time, outcome, events, webhook_duration_ms, response_status, response_body, error) VALUES (?, ?, ?, 'transient_error', 7, 91, 503, 'unavailable', 'retry')", workspaceID, id, timeMs))
	}
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/"+id+"#read")
	type delivery struct {
		Time, Events, DurationMs, ResponseStatus int64
		Outcome, ResponseBody, Error             string
	}
	type response struct{ Data []delivery }
	result := testutil.CallRoute[openapi.LogdrainIdRequest, response](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.LogdrainIdRequest{LogdrainId: id})
	require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
	require.Len(t, result.Body.Data, 20)
	require.Equal(t, delivery{Time: now, Events: 7, DurationMs: 91, ResponseStatus: 503, Outcome: "transient_error", ResponseBody: "unavailable", Error: "retry"}, result.Body.Data[0])
	require.Equal(t, now-19, result.Body.Data[19].Time)
}
