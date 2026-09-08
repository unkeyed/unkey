package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// Cursor identifies an event's position in insertion order.
type Cursor struct {
	Time    int64
	EventID string
}

// Source reads one stream of events for export.
type Source interface {
	// Read returns events after from with inserted_at before toExclusive,
	// ordered by (inserted_at, event_id), and capped at limit rows. The returned
	// cursor identifies the last row; empty results and errors return from unchanged.
	Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int) ([]sink.Event, Cursor, error)
}

// AuditLogs reads the audit_logs stream from ClickHouse.
type AuditLogs struct{ client *clickhouse.Client }

// NewAuditLogs binds the source to the ClickHouse client used for export reads.
func NewAuditLogs(client *clickhouse.Client) *AuditLogs { return &AuditLogs{client: client} }

// rawJSON wraps a JSON string from ClickHouse as json.RawMessage, mapping
// empty strings to JSON null so the payload stays valid JSON.
func rawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("null")
	}
	return json.RawMessage(s)
}

// auditRow is a read-side row for audit_logs_raw_v1. The table stores
// actor_meta, meta, and targets.meta as native JSON columns, which
// clickhouse-go cannot scan into json.RawMessage. The query stringifies
// them with toJSONString so they scan into plain strings.
type auditRow struct {
	EventID     string   `ch:"event_id"`
	Time        int64    `ch:"time"`
	InsertedAt  int64    `ch:"inserted_at"`
	Event       string   `ch:"event"`
	Description string   `ch:"description"`
	ActorType   string   `ch:"actor_type"`
	ActorID     string   `ch:"actor_id"`
	ActorName   string   `ch:"actor_name"`
	ActorMeta   string   `ch:"actor_meta_json"`
	RemoteIP    string   `ch:"remote_ip"`
	UserAgent   string   `ch:"user_agent"`
	Meta        string   `ch:"meta_json"`
	TargetTypes []string `ch:"target_types"`
	TargetIDs   []string `ch:"target_ids"`
	TargetNames []string `ch:"target_names"`
	TargetMetas []string `ch:"target_metas_json"`

	CorrelationID string `ch:"correlation_id"`
}

// Read preserves deterministic timestamp paging while converting ClickHouse rows into the public audit-log shape.
func (s *AuditLogs) Read(ctx context.Context, workspaceID string, from Cursor, toExclusive int64, limit int) ([]sink.Event, Cursor, error) {
	// query orders by inserted_at and event_id so paging stays deterministic
	// when many rows share one inserted_at millisecond.
	const query = `
		SELECT
			event_id,
			time,
			inserted_at,
			event,
			description,
			actor_type,
			actor_id,
			actor_name,
			toJSONString(actor_meta) AS actor_meta_json,
			remote_ip,
			user_agent,
			toJSONString(meta) AS meta_json,
			targets.type AS target_types,
			targets.id AS target_ids,
			targets.name AS target_names,
			arrayMap(x -> toJSONString(x), targets.meta) AS target_metas_json,
			correlation_id
		FROM audit_logs_raw_v1
		WHERE workspace_id = {workspace:String}
			AND (inserted_at, event_id) > ({from_time:Int64}, {from_id:String})
			AND inserted_at < {to:Int64}
		ORDER BY inserted_at, event_id
		LIMIT {batch_size:UInt64}`
	// The parameter is named batch_size because a query parameter named
	// "limit" collides with the ClickHouse server setting of the same name
	// and fails with CANNOT_PARSE_QUOTED_STRING.
	rows, err := clickhouse.Select[auditRow](ctx, s.client.Conn(), query, map[string]string{
		"workspace":  workspaceID,
		"from_time":  strconv.FormatInt(from.Time, 10),
		"from_id":    from.EventID,
		"to":         strconv.FormatInt(toExclusive, 10),
		"batch_size": strconv.Itoa(limit),
	})
	if err != nil {
		return nil, from, fmt.Errorf("read audit logs: %w", err)
	}
	events := make([]sink.Event, 0, len(rows))
	next := from
	for _, row := range rows {
		// targets is a ClickHouse Nested column, so the four arrays always
		// have equal length.
		targets := make([]sink.AuditLogTarget, len(row.TargetIDs))
		for i := range row.TargetIDs {
			targets[i] = sink.AuditLogTarget{
				ID:       row.TargetIDs[i],
				Type:     row.TargetTypes[i],
				Name:     row.TargetNames[i],
				Metadata: rawJSON(row.TargetMetas[i]),
			}
		}
		actor := sink.AuditLogActor{
			ID:       row.ActorID,
			Type:     row.ActorType,
			Name:     row.ActorName,
			Metadata: rawJSON(row.ActorMeta),
		}
		payload := sink.AuditLogPayload{
			ID:            row.EventID,
			Action:        row.Event,
			OccurredAt:    sink.FormatTime(row.Time),
			Actor:         actor,
			Targets:       targets,
			Context:       sink.AuditLogContext{Location: row.RemoteIP, UserAgent: row.UserAgent},
			Metadata:      rawJSON(row.Meta),
			Description:   row.Description,
			CorrelationID: row.CorrelationID,
		}
		events = append(events, sink.Event{EventID: row.EventID, Stream: "audit_logs", Time: row.Time, Payload: payload})
		next = Cursor{Time: row.InsertedAt, EventID: row.EventID}
	}
	return events, next, nil
}
