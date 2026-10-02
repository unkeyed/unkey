package source_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

func TestRuntimeLogsRead_Payload(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New("workspace")
	now := time.Now().UnixMilli()
	require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO runtime_logs_raw_v1
		(workspace_id, log_id, time, severity, message, attributes, project_id, app_id,
		environment_id, deployment_id, region, k8s_pod_name, platform)
		VALUES (?, 'rlog_1', ?, 'fatal', 'Payment failed', ?,
		'project_1', 'app_1', 'env_1', 'deployment_1', 'eu-west-1', 'internal-pod', 'aws')`, workspace, now-3600000, `{"order":{"id":42},"retry":false}`))
	events, cursor, err := source.NewRuntimeLogs(client).Read(t.Context(), workspace, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "runtime_logs", events[0].Stream)
	require.Equal(t, now-3600000, events[0].Time)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, "rlog_1", cursor.EventID)
	require.Equal(t, sink.RuntimeLogPayload{
		LogID:         "rlog_1",
		Severity:      "fatal",
		Message:       "Payment failed",
		Attributes:    json.RawMessage(`{"order":{"id":42},"retry":false}`),
		ProjectID:     "project_1",
		AppID:         "app_1",
		EnvironmentID: "env_1",
		DeploymentID:  "deployment_1",
		Region:        "eu-west-1",
	}, events[0].Payload)
}

func TestRuntimeLogsRead_FilteredCursorBounds(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New("workspace")
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		workspace, id, project, app, environment, severity string
		insertedAt                                         int64
	}{
		{workspace, "a", "other", "app", "env", "error", now},
		{workspace, "b", "project", "other", "env", "error", now},
		{workspace, "c", "project", "app", "other", "error", now},
		{workspace, "d", "project", "app", "env", "info", now},
		{uid.New("workspace"), "e", "project", "app", "env", "error", now},
		{workspace, "f", "project", "app", "env", "error", now},
		{workspace, "g", "project", "app2", "env2", "warn", now},
		{workspace, "a", "project", "app", "env", "error", now + 1},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO runtime_logs_raw_v1
			(workspace_id, log_id, inserted_at, time, project_id, app_id, environment_id, severity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, row.workspace, row.id, row.insertedAt, now-60000, row.project, row.app, row.environment, row.severity))
	}
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_RuntimeLogs{RuntimeLogs: &logdrainv1.RuntimeLogStreamConfig{
		Severities: []string{"error", "warn"}, ProjectIds: []string{"project"}, AppIds: []string{"app", "app2"}, EnvironmentIds: []string{"env", "env2"},
	}}}
	reader := source.NewRuntimeLogs(client)
	page, next, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "f", page[0].EventID)
	page, next, err = reader.Read(t.Context(), workspace, next, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "g", page[0].EventID)
	require.Equal(t, source.Cursor{Time: now, EventID: "g"}, next)
	page, final, err := reader.Read(t.Context(), workspace, next, now+1, 1, filter)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Equal(t, next, final)
	page, next, err = reader.Read(t.Context(), workspace, next, now+2, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, source.Cursor{Time: now + 1, EventID: "a"}, next)
	page, _, err = reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, 1, nil)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "a", page[0].EventID)
}
