package harness

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// SeedSentinelRequest inserts a synthetic row into
// default.sentinel_requests_raw_v1 that looks as if real sentinel
// handled a probe request. Used by tier-1.11 (request_logs) tests in
// dev, where no sentinel exists to write the row for real.
//
// Path is set to "/preflight-<runID>" so the probe's query filter
// round-trips deterministically without needing to inspect the
// generated request_id.
//
// The row fields match what svc/sentinel/middleware/logging.go would
// actually write. If those fields ever drift, the probe's query will
// stop finding seeded rows; that is the signal to update both.
func (h *Harness) SeedSentinelRequest(t *testing.T, ctx context.Context, runID string) {
	t.Helper()
	row := schema.SentinelRequest{
		RequestID:       "preflight-req-" + runID,
		Time:            time.Now().UnixMilli(),
		WorkspaceID:     h.Seed.Resources.UserWorkspace.ID,
		EnvironmentID:   "",
		ProjectID:       "",
		SentinelID:      "preflight-seeded",
		DeploymentID:    "",
		InstanceID:      "preflight-seeded",
		InstanceAddress: "127.0.0.1",
		Region:          h.Region,
		Platform:        "harness",
		Method:          "GET",
		Host:            "preflight.local",
		Path:            "/preflight-" + runID,
		QueryString:     "",
		QueryParams:     map[string][]string{},
		RequestHeaders:  []string{"X-Preflight-Run-Id: " + runID},
		RequestBody:     "",
		ResponseStatus:  200,
		ResponseHeaders: []string{},
		ResponseBody:    "",
		UserAgent:       "preflight-probe",
		IPAddress:       "127.0.0.1",
		TotalLatency:    1,
		InstanceLatency: 0,
		SentinelLatency: 1,
	}
	if h.Project != nil {
		row.ProjectID = h.Project.ID
	}
	if h.Environment != nil {
		row.EnvironmentID = h.Environment.ID
	}
	if h.App != nil {
		row.DeploymentID = "dep_preflight_" + h.App.ID
	}

	batch, err := h.ClickHouseConn.PrepareBatch(ctx, "INSERT INTO default.sentinel_requests_raw_v1")
	require.NoError(t, err)
	require.NoError(t, batch.AppendStruct(&row))
	require.NoError(t, batch.Send())
}
