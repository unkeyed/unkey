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

const gatewayBatchBytesMax = 16 << 20

func (s *GatewayRequests) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int, cfg *logdrainv1.Config) ([]sink.Event, Cursor, error) {
	const columns = `inserted_at, time, request_id, project_id, app_id,
		environment_id, deployment_id, region, method, host, path, response_status,
		total_latency, instance_latency, gateway_latency, query_string, query_params,
		request_headers, request_body, response_headers, response_body, user_agent, ip_address`
	// JSON escaping can expand one byte to six. The remaining charge covers field
	// names and both destination envelopes. Limit bytes before the driver decodes rows.
	const candidates = `SELECT ` + columns + `,
		6 * length(toJSONString(tuple(` + columns + `))) + 1024 AS size_bytes
		FROM frontline_requests_raw_v1
		WHERE workspace_id = {workspace:String}
		AND (inserted_at > {from_time:Int64}
		  OR (inserted_at = {from_time:Int64} AND request_id > {from_id:String}))
		AND inserted_at < {to:Int64}
		AND (empty({statuses:Array(Int32)}) OR intDiv(response_status, 100) IN {statuses:Array(Int32)})
		AND (empty({projects:Array(String)}) OR project_id IN {projects:Array(String)})
		AND (empty({apps:Array(String)}) OR app_id IN {apps:Array(String)})
		AND (empty({environments:Array(String)}) OR environment_id IN {environments:Array(String)})
		ORDER BY inserted_at, request_id LIMIT {candidates_max:UInt64} WITH TIES`
	// Peer-inclusive windows keep an identical cursor group indivisible at either limit.
	const query = `SELECT ` + columns + ` FROM (
		SELECT ` + columns + `, sum(size_bytes) OVER w AS size_bytes_sum, count() OVER w AS rows_sum
		FROM (` + candidates + `)
		WINDOW w AS (ORDER BY inserted_at, request_id RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)
	) WHERE size_bytes_sum <= {bytes_max:UInt64} AND rows_sum <= {batch_size:UInt64}
	ORDER BY inserted_at, request_id`
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
	parameters := map[string]string{
		"workspace":      workspaceID,
		"from_time":      strconv.FormatInt(from.Time, 10),
		"from_id":        from.EventID,
		"to":             strconv.FormatInt(toExclusive, 10),
		"batch_size":     strconv.Itoa(limit),
		"candidates_max": strconv.Itoa(min(limit, gatewayBatchBytesMax/1024)),
		"bytes_max":      strconv.Itoa(gatewayBatchBytesMax),
		"statuses":       "[" + strings.Join(statuses, ",") + "]",
		"projects":       clickhouse.StringArrayParam(filters.GetProjectIds()),
		"apps":           clickhouse.StringArrayParam(filters.GetAppIds()),
		"environments":   clickhouse.StringArrayParam(filters.GetEnvironmentIds()),
	}
	rows, err := clickhouse.Select[row](ctx, s.client.Conn(), query, parameters)
	if err != nil {
		return nil, from, fmt.Errorf("read gateway requests: %w", err)
	}
	if len(rows) == 0 {
		const probe = `SELECT inserted_at, left(request_id, 128) AS request_id_prefix,
			sum(size_bytes) AS size_bytes, count() AS rows_count
			FROM (` + candidates + `) GROUP BY inserted_at, request_id
			ORDER BY inserted_at, request_id LIMIT 1`
		type group struct {
			InsertedAt      int64  `ch:"inserted_at"`
			RequestIDPrefix string `ch:"request_id_prefix"`
			SizeBytes       uint64 `ch:"size_bytes"`
			RowsCount       uint64 `ch:"rows_count"`
		}
		groups, err := clickhouse.Select[group](ctx, s.client.Conn(), probe, parameters)
		if err != nil {
			return nil, from, fmt.Errorf("check gateway window exhaustion: %w", err)
		}
		if len(groups) > 0 {
			first := groups[0]
			if first.SizeBytes > gatewayBatchBytesMax || first.RowsCount > uint64(limit) {
				return nil, from, fmt.Errorf("gateway cursor group exceeds batch limits at (%d, %q): estimated bytes %d/%d, rows %d/%d; cursor retained",
					first.InsertedAt, first.RequestIDPrefix, first.SizeBytes, gatewayBatchBytesMax, first.RowsCount, limit)
			}
			return nil, from, fmt.Errorf("gateway page changed while checking exhaustion; retry with cursor retained")
		}
		return nil, from, nil
	}
	events := make([]sink.Event, 0, len(rows))
	next := from
	for _, row := range rows {
		payload := sink.GatewayRequestPayload{
			RequestID:     row.RequestID,
			ProjectID:     row.ProjectID,
			AppID:         row.AppID,
			EnvironmentID: row.EnvironmentID,
			DeploymentID:  row.DeploymentID,
			Region:        row.Region,
			Request: sink.GatewayRequest{
				Method:      row.Method,
				Host:        row.Host,
				Path:        row.Path,
				QueryString: row.QueryString,
				QueryParams: row.QueryParams,
				Headers:     row.RequestHeaders,
				Body:        row.RequestBody,
				UserAgent:   row.UserAgent,
				IPAddress:   row.IPAddress,
			},
			Response: sink.GatewayResponse{Status: row.ResponseStatus, Headers: row.ResponseHeaders, Body: row.ResponseBody},
			Latency:  sink.GatewayRequestLatency{Total: row.TotalLatency, Instance: row.InstanceLatency, Gateway: row.GatewayLatency},
		}
		events = append(events, sink.Event{EventID: row.RequestID, Stream: "gateway_requests", Time: row.Time, Payload: payload})
		next = Cursor{Time: row.InsertedAt, EventID: row.RequestID}
	}
	return events, next, nil
}
