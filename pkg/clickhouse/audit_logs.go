package clickhouse

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/fault"
)

// InsertAuditLogs writes a batch of audit log rows to audit_logs_raw_v1 and
// returns only after ClickHouse confirms the insert. The outbox worker relies
// on this synchronous confirmation before marking the source MySQL rows as
// exported; on CH failure the caller retries, and the insert block's content
// hash lets ClickHouse's non_replicated_deduplication_window drop identical
// retries as a noop.
func (c *Client) InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error {
	if len(rows) == 0 {
		return nil
	}
	return flush(c, ctx, rows)
}

// ListAuditLogsRequest holds the parameters for a single page read of
// audit_logs_raw_v1. All filters are optional except WorkspaceID and the time
// window. The time window and the keyset cursor operate on inserted_at (the
// drain-stamped visibility timestamp), NOT the event time, so that late-drained
// rows are never skipped by an advancing consumer watermark. See the
// /v2/audit.getLogs plan (KTD3) for why event time would produce silent gaps.
type ListAuditLogsRequest struct {
	WorkspaceID string
	Bucket      string
	// StartMs/EndMs bound inserted_at (inclusive), in unix milliseconds.
	StartMs int64
	EndMs   int64

	// AfterInsertedAtMs/AfterEventID form the keyset cursor. When
	// AfterEventID is empty the query starts at the beginning of the window.
	AfterInsertedAtMs int64
	AfterEventID      string

	// Optional filters.
	Events       []string
	ActorID      string
	ResourceType string

	// Limit is the number of rows to return. Callers typically pass limit+1 to
	// detect whether another page exists.
	Limit int
}

// auditLogReadRow mirrors AuditLogV1 but scans the JSON columns as plain
// strings. The meta columns (actor_meta, meta, targets.meta) are native
// ClickHouse JSON, which we read back via toJSONString(...) so the driver hands
// us bytes we can pass through verbatim instead of a decoded type. Nested
// subcolumns are aliased to dot-free names so the `ch` struct tags resolve.
type auditLogReadRow struct {
	EventID     string `ch:"event_id"`
	Time        int64  `ch:"time"`
	InsertedAt  int64  `ch:"inserted_at"`
	WorkspaceID string `ch:"workspace_id"`
	Bucket      string `ch:"bucket"`
	Source      string `ch:"source"`

	Event       string `ch:"event"`
	Description string `ch:"description"`

	ActorType string `ch:"actor_type"`
	ActorID   string `ch:"actor_id"`
	ActorName string `ch:"actor_name"`
	ActorMeta string `ch:"actor_meta"`

	RemoteIP  string `ch:"remote_ip"`
	UserAgent string `ch:"user_agent"`
	Meta      string `ch:"meta"`

	TargetTypes []string `ch:"targets_type"`
	TargetIDs   []string `ch:"targets_id"`
	TargetNames []string `ch:"targets_name"`
	TargetMetas []string `ch:"targets_meta"`

	CorrelationID string `ch:"correlation_id"`
}

// listAuditLogsColumns is the SELECT projection. The JSON columns are rendered
// with toJSONString(...) and the Nested array is mapped element-wise, all
// aliased to the dot-free names auditLogReadRow expects.
const listAuditLogsColumns = `event_id, time, inserted_at, workspace_id, bucket, source,
	event, description,
	actor_type, actor_id, actor_name, toJSONString(actor_meta) AS actor_meta,
	remote_ip, user_agent, toJSONString(meta) AS meta,
	` + "`targets.type`" + ` AS targets_type,
	` + "`targets.id`" + ` AS targets_id,
	` + "`targets.name`" + ` AS targets_name,
	arrayMap(x -> toJSONString(x), ` + "`targets.meta`" + `) AS targets_meta,
	correlation_id`

// ListAuditLogs reads a single ascending page of audit events for a workspace,
// ordered and keyset-paginated on (inserted_at, event_id). The query injects
// workspace_id itself; callers never supply it as untrusted input. It returns
// typed AuditLogV1 rows with the JSON meta columns preserved as raw bytes.
func (c *Client) ListAuditLogs(ctx context.Context, req ListAuditLogsRequest) ([]schema.AuditLogV1, error) {
	params := map[string]string{
		"workspaceID": req.WorkspaceID,
		"bucket":      req.Bucket,
		"startMs":     strconv.FormatInt(req.StartMs, 10),
		"endMs":       strconv.FormatInt(req.EndMs, 10),
		"limit":       strconv.Itoa(req.Limit),
	}

	var where strings.Builder
	where.WriteString(`workspace_id = {workspaceID:String}
		AND bucket = {bucket:String}
		AND inserted_at BETWEEN {startMs:Int64} AND {endMs:Int64}`)

	if len(req.Events) > 0 {
		where.WriteString(` AND event IN {events:Array(String)}`)
		params["events"] = stringArrayParam(req.Events)
	}
	if req.ActorID != "" {
		where.WriteString(` AND actor_id = {actorID:String}`)
		params["actorID"] = req.ActorID
	}
	if req.ResourceType != "" {
		where.WriteString(" AND has(`targets.type`, {resourceType:String})")
		params["resourceType"] = req.ResourceType
	}
	// Keyset cursor: strictly after (inserted_at, event_id). Only applied once a
	// page has been read (AfterEventID set), so the first page starts at the
	// window boundary.
	if req.AfterEventID != "" {
		where.WriteString(` AND (inserted_at > {afterTs:Int64}
			OR (inserted_at = {afterTs:Int64} AND event_id > {afterID:String}))`)
		params["afterTs"] = strconv.FormatInt(req.AfterInsertedAtMs, 10)
		params["afterID"] = req.AfterEventID
	}

	query := `SELECT ` + listAuditLogsColumns + `
		FROM default.audit_logs_raw_v1
		WHERE ` + where.String() + `
		ORDER BY inserted_at ASC, event_id ASC
		LIMIT {limit:Int32}`

	rows, err := Select[auditLogReadRow](ctx, c.conn, query, params)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to list audit logs"))
	}

	out := make([]schema.AuditLogV1, len(rows))
	for i, r := range rows {
		out[i] = schema.AuditLogV1{
			EventID:       r.EventID,
			Time:          r.Time,
			InsertedAt:    r.InsertedAt,
			WorkspaceID:   r.WorkspaceID,
			Bucket:        r.Bucket,
			Source:        r.Source,
			Event:         r.Event,
			Description:   r.Description,
			ActorType:     r.ActorType,
			ActorID:       r.ActorID,
			ActorName:     r.ActorName,
			ActorMeta:     json.RawMessage(r.ActorMeta),
			RemoteIP:      r.RemoteIP,
			UserAgent:     r.UserAgent,
			Meta:          json.RawMessage(r.Meta),
			TargetTypes:   r.TargetTypes,
			TargetIDs:     r.TargetIDs,
			TargetNames:   r.TargetNames,
			TargetMetas:   toRawMessages(r.TargetMetas),
			CorrelationID: r.CorrelationID,
		}
	}
	return out, nil
}

func toRawMessages(ss []string) []json.RawMessage {
	if len(ss) == 0 {
		return nil
	}
	out := make([]json.RawMessage, len(ss))
	for i, s := range ss {
		out[i] = json.RawMessage(s)
	}
	return out
}
