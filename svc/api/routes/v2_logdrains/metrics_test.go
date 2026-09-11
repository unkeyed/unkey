package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestMetricsAggregateOnlyThisDrainAndWorkspace(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &logdrains.Metrics{DB: h.DB, ClickHouse: h.ClickHouse, Clock: h.Clock}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	_, err := h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Metrics', 'audit_logs', '', '', '', 123)", id, workspaceID)
	require.NoError(t, err)
	bucket := h.Clock.Now().UnixMilli()/60_000*60_000 - 60_000
	for i, outcome := range []string{"success", "transient_error", "success", "permanent_error"} {
		workspace := workspaceID
		if i == 3 {
			workspace = uid.New("ws")
		}
		require.NoError(t, h.ClickHouse.Exec(context.Background(), "INSERT INTO default.logdrain_deliveries_raw_v1 (workspace_id, drain_id, time, outcome, events, webhook_duration_ms) VALUES (?, ?, ?, ?, ?, ?)", workspace, id, bucket+int64(i)+1, outcome, int64(7+i*2), int64(10+i*20)))
	}
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/"+id+"#read")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	type point struct {
		Ts, SuccessCount, TransientErrorCount, PermanentErrorCount, EventsDelivered, LastSuccessMs int64
		AvgDurationMs                                                                              float64
	}
	type response struct {
		Data struct {
			Series        []point
			BucketMinutes int
		}
	}
	result := testutil.CallRoute[json.RawMessage, response](h, route, headers, json.RawMessage(`{"logdrainId":"`+id+`","hours":1}`))
	require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
	require.Equal(t, 1, result.Body.Data.BucketMinutes)
	require.Len(t, result.Body.Data.Series, 61)
	var matching []point
	for _, point := range result.Body.Data.Series {
		if point.Ts == bucket {
			matching = append(matching, point)
		}
	}
	require.Equal(t, []point{{Ts: bucket, SuccessCount: 2, TransientErrorCount: 1, PermanentErrorCount: 0, EventsDelivered: 18, LastSuccessMs: bucket + 3, AvgDurationMs: 30}}, matching)
}
