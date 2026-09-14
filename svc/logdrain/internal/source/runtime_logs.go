package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// RuntimeLogs reads application output in insertion order.
type RuntimeLogs struct{ client *clickhouse.Client }

func NewRuntimeLogs(client *clickhouse.Client) *RuntimeLogs {
	return &RuntimeLogs{client: client}
}

func (s *RuntimeLogs) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int, cfg *logdrainv1.Config) ([]sink.Event, Cursor, error) {
	const query = `SELECT inserted_at, time, log_id, severity, message,
		toJSONString(attributes) AS attributes_json, project_id, app_id,
		environment_id, deployment_id, region FROM runtime_logs_raw_v1
		WHERE workspace_id = {workspace:String}
		AND (inserted_at > {from_time:Int64}
		  OR (inserted_at = {from_time:Int64} AND log_id > {from_id:String}))
		AND inserted_at < {to:Int64}
		AND (empty({severities:Array(String)}) OR severity IN {severities:Array(String)})
		AND (empty({projects:Array(String)}) OR project_id IN {projects:Array(String)})
		AND (empty({apps:Array(String)}) OR app_id IN {apps:Array(String)})
		AND (empty({environments:Array(String)}) OR environment_id IN {environments:Array(String)})
		ORDER BY inserted_at, log_id LIMIT {batch_size:UInt64}`
	type row struct {
		InsertedAt    int64  `ch:"inserted_at"`
		Time          int64  `ch:"time"`
		LogID         string `ch:"log_id"`
		Severity      string `ch:"severity"`
		Message       string `ch:"message"`
		Attributes    string `ch:"attributes_json"`
		ProjectID     string `ch:"project_id"`
		AppID         string `ch:"app_id"`
		EnvironmentID string `ch:"environment_id"`
		DeploymentID  string `ch:"deployment_id"`
		Region        string `ch:"region"`
	}
	filters := cfg.GetRuntimeLogs()
	rows, err := clickhouse.Select[row](ctx, s.client.Conn(), query, map[string]string{
		"workspace":    workspaceID,
		"from_time":    strconv.FormatInt(from.Time, 10),
		"from_id":      from.EventID,
		"to":           strconv.FormatInt(toExclusive, 10),
		"batch_size":   strconv.Itoa(limit),
		"severities":   clickhouse.StringArrayParam(filters.GetSeverities()),
		"projects":     clickhouse.StringArrayParam(filters.GetProjectIds()),
		"apps":         clickhouse.StringArrayParam(filters.GetAppIds()),
		"environments": clickhouse.StringArrayParam(filters.GetEnvironmentIds()),
	})
	if err != nil {
		return nil, from, fmt.Errorf("read runtime logs: %w", err)
	}
	events := make([]sink.Event, 0, len(rows))
	next := from
	for _, row := range rows {
		payload := sink.RuntimeLogPayload{
			LogID: row.LogID, Severity: row.Severity, Message: row.Message,
			Attributes: json.RawMessage(row.Attributes), ProjectID: row.ProjectID,
			AppID: row.AppID, EnvironmentID: row.EnvironmentID,
			DeploymentID: row.DeploymentID, Region: row.Region,
		}
		events = append(events, sink.Event{EventID: row.LogID, Stream: "runtime_logs", Time: row.Time, Payload: payload})
		next = Cursor{Time: row.InsertedAt, EventID: row.LogID}
	}
	return events, next, nil
}
