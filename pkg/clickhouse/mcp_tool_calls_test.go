package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestMcpToolCallsRollUpHourly guarantees that svc/mcp's rows insert with the
// generated column list and that the hourly rollup counts calls per outcome
// and distinct workspaces only from rows that carry one.
func TestMcpToolCallsRollUpHourly(t *testing.T) {
	client, err := New(Config{URL: containers.ClickHouse(t).DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	tool := "unkey_v2_" + uid.New(uid.TestPrefix)
	workspaceID := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	row := func(outcome string, status int32, workspace string, latencyMs int64) schema.McpToolCallV1 {
		return schema.McpToolCallV1{
			RequestID:     uid.New(uid.RequestPrefix),
			Time:          now,
			Toolset:       "v2",
			Mount:         "/mcp/v2",
			Tool:          tool,
			Outcome:       outcome,
			HttpStatus:    status,
			ErrorType:     "",
			LatencyMs:     latencyMs,
			AuthKind:      "oauth",
			WorkspaceID:   workspace,
			SessionID:     uid.New(uid.CLISessionPrefix),
			ApiRequestID:  uid.New(uid.RequestPrefix),
			ClientName:    "claude-code",
			ClientVersion: "2.1.0",
		}
	}
	rows := []schema.McpToolCallV1{
		row("ok", 200, workspaceID, 10),
		row("ok", 200, workspaceID, 30),
		row("api_error", 403, "", 20),
	}
	require.NoError(t, flush(client, t.Context(), rows))

	var stored []schema.McpToolCallV1
	query := "SELECT " + rows[0].InsertColumns() + " FROM " + rows[0].Table() + " WHERE tool = ? ORDER BY latency_ms"
	require.NoError(t, client.conn.Select(t.Context(), &stored, query, tool))
	require.Equal(t, []schema.McpToolCallV1{rows[0], rows[2], rows[1]}, stored)

	var hourly []struct {
		Outcome    string `ch:"outcome"`
		Count      int64  `ch:"count"`
		Workspaces uint64 `ch:"workspaces"`
	}
	require.NoError(t, client.conn.Select(t.Context(), &hourly, `
		SELECT outcome, sum(count) AS count, uniqIfMerge(workspaces) AS workspaces
		FROM default.mcp_tool_calls_per_hour_v1
		WHERE tool = ?
		GROUP BY outcome
		ORDER BY outcome`, tool))
	require.Len(t, hourly, 2)
	require.Equal(t, "api_error", hourly[0].Outcome)
	require.Equal(t, int64(1), hourly[0].Count)
	require.Equal(t, uint64(0), hourly[0].Workspaces)
	require.Equal(t, "ok", hourly[1].Outcome)
	require.Equal(t, int64(2), hourly[1].Count)
	require.Equal(t, uint64(1), hourly[1].Workspaces)
}
