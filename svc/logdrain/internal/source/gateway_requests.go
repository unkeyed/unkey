package source

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// GatewayRequests reads gateway HTTP requests in insertion order.
type GatewayRequests struct{ client *clickhouse.Client }

func NewGatewayRequests(client *clickhouse.Client) *GatewayRequests {
	return &GatewayRequests{client: client}
}

func (s *GatewayRequests) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int, cfg *logdrainv1.Config) ([]sink.Event, Cursor, error) {
	const query = `SELECT inserted_at, time, request_id, project_id, app_id,
		environment_id, deployment_id, region, method, host, path, response_status,
		total_latency, instance_latency, gateway_latency, query_string, query_params,
		request_headers, request_body, response_headers, response_body, user_agent, ip_address
		FROM frontline_requests_raw_v1
		WHERE workspace_id = {workspace:String}
		AND (inserted_at > {from_time:Int64}
		  OR (inserted_at = {from_time:Int64} AND request_id > {from_id:String}))
		AND inserted_at < {to:Int64}
		AND (empty({statuses:Array(Int32)}) OR intDiv(response_status, 100) IN {statuses:Array(Int32)})
		AND (empty({projects:Array(String)}) OR project_id IN {projects:Array(String)})
		AND (empty({apps:Array(String)}) OR app_id IN {apps:Array(String)})
		AND (empty({environments:Array(String)}) OR environment_id IN {environments:Array(String)})
		ORDER BY inserted_at, request_id LIMIT {batch_size:UInt64}`
	type row struct {
		InsertedAt int64 `ch:"inserted_at"`
		schema.FrontlineRequest
	}
	filters := cfg.GetGatewayRequests()
	statuses := make([]string, len(filters.GetStatusClasses()))
	for i, status := range filters.GetStatusClasses() {
		switch status {
		case logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_2XX,
			logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_3XX,
			logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_4XX,
			logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_5XX:
			statuses[i] = strconv.FormatInt(int64(status), 10)
		case logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_UNSPECIFIED:
			return nil, from, fmt.Errorf("status class must be between 2 and 5: %d", status)
		default:
			return nil, from, fmt.Errorf("status class must be between 2 and 5: %d", status)
		}
	}
	rows, err := clickhouse.Select[row](ctx, s.client.Conn(), query, map[string]string{
		"workspace":    workspaceID,
		"from_time":    strconv.FormatInt(from.Time, 10),
		"from_id":      from.EventID,
		"to":           strconv.FormatInt(toExclusive, 10),
		"batch_size":   strconv.Itoa(limit),
		"statuses":     "[" + strings.Join(statuses, ",") + "]",
		"projects":     clickhouse.StringArrayParam(filters.GetProjectIds()),
		"apps":         clickhouse.StringArrayParam(filters.GetAppIds()),
		"environments": clickhouse.StringArrayParam(filters.GetEnvironmentIds()),
	})
	if err != nil {
		return nil, from, fmt.Errorf("read gateway requests: %w", err)
	}
	events := make([]sink.Event, 0, len(rows))
	next := from
	for _, row := range rows {
		payload := sink.GatewayRequestPayload{
			RequestID:       row.RequestID,
			ProjectID:       row.ProjectID,
			AppID:           row.AppID,
			EnvironmentID:   row.EnvironmentID,
			DeploymentID:    row.DeploymentID,
			Region:          row.Region,
			Method:          row.Method,
			Host:            row.Host,
			Path:            row.Path,
			QueryString:     row.QueryString,
			QueryParams:     row.QueryParams,
			RequestHeaders:  row.RequestHeaders,
			RequestBody:     row.RequestBody,
			ResponseHeaders: row.ResponseHeaders,
			ResponseBody:    row.ResponseBody,
			UserAgent:       row.UserAgent,
			IPAddress:       row.IPAddress,
			ResponseStatus:  row.ResponseStatus,
			TotalLatency:    row.TotalLatency,
			InstanceLatency: row.InstanceLatency,
			GatewayLatency:  row.GatewayLatency,
		}
		events = append(events, sink.Event{EventID: row.RequestID, Stream: "gateway_requests", Time: row.Time, Payload: payload})
		next = Cursor{Time: row.InsertedAt, EventID: row.RequestID}
	}
	return events, next, nil
}
