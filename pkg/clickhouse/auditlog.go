package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// NewAuditLogBuffer buffers canonical audit events and converts them to the
// ClickHouse storage shape during each background flush. Callers can use
// [auditlog.EventTarget] values without managing parallel Nested arrays.
func NewAuditLogBuffer(c *Client, cfg BufferConfig) *batch.BatchProcessor[auditlog.Event] {
	var row schema.AuditLogV1
	table := row.Table()
	onErr := flushErrorHandler(cfg)

	return batch.New(batch.Config[auditlog.Event]{
		Name:          cfg.Name,
		Drop:          cfg.Drop,
		BatchSize:     cfg.BatchSize,
		BufferSize:    cfg.BufferSize,
		FlushInterval: cfg.FlushInterval,
		Consumers:     cfg.Consumers,
		Flush: func(ctx context.Context, events []auditlog.Event) {
			rows, err := EncodeAuditLogEvents(events)
			if err != nil {
				onErr(ctx, table, len(events), err)
				return
			}
			if err := flush(c, ctx, rows); err != nil {
				onErr(ctx, table, len(rows), err)
			}
		},
	})
}

// EncodeAuditLogEvents converts canonical audit events to ClickHouse rows.
// ClickHouse assigns inserted_at, so the conversion does not require an insert
// timestamp from the caller.
func EncodeAuditLogEvents(events []auditlog.Event) ([]schema.AuditLogV1, error) {
	rows := make([]schema.AuditLogV1, len(events))
	for i, event := range events {
		row, err := encodeAuditLogEvent(event)
		if err != nil {
			return nil, err
		}
		rows[i] = row
	}
	return rows, nil
}

// encodeAuditLogEvent converts one event and preserves metadata encoding errors.
func encodeAuditLogEvent(event auditlog.Event) (schema.AuditLogV1, error) {
	actorMeta, err := encodeAuditLogMeta(event.Actor.Meta)
	if err != nil {
		return schema.AuditLogV1{}, fmt.Errorf("encode actor_meta event_id=%s: %w", event.EventID, err)
	}
	meta, err := encodeAuditLogMeta(event.Meta)
	if err != nil {
		return schema.AuditLogV1{}, fmt.Errorf("encode meta event_id=%s: %w", event.EventID, err)
	}

	targetTypes := make([]string, len(event.Targets))
	targetIDs := make([]string, len(event.Targets))
	targetNames := make([]string, len(event.Targets))
	targetMetas := make([]json.RawMessage, len(event.Targets))
	for i, target := range event.Targets {
		targetTypes[i] = target.Type
		targetIDs[i] = target.ID
		targetNames[i] = target.Name

		targetMeta, encodeErr := encodeAuditLogMeta(target.Meta)
		if encodeErr != nil {
			return schema.AuditLogV1{}, fmt.Errorf("encode target_meta event_id=%s target_id=%s: %w", event.EventID, target.ID, encodeErr)
		}
		targetMetas[i] = targetMeta
	}

	source := event.Source
	if source == "" {
		source = auditlog.EventSourcePlatform
	}

	return schema.AuditLogV1{
		EventID:       event.EventID,
		Time:          event.Time,
		WorkspaceID:   event.WorkspaceID,
		Bucket:        event.Bucket,
		Source:        source,
		Event:         event.Event,
		Description:   event.Description,
		ActorType:     event.Actor.Type,
		ActorID:       event.Actor.ID,
		ActorName:     event.Actor.Name,
		ActorMeta:     actorMeta,
		RemoteIP:      event.RemoteIP,
		UserAgent:     event.UserAgent,
		Meta:          meta,
		TargetTypes:   targetTypes,
		TargetIDs:     targetIDs,
		TargetNames:   targetNames,
		TargetMetas:   targetMetas,
		CorrelationID: event.CorrelationID,
	}, nil
}

// encodeAuditLogMeta returns an object for a ClickHouse JSON column. Empty
// metadata becomes {} because some ClickHouse configurations reject raw nulls.
func encodeAuditLogMeta(meta map[string]any) (json.RawMessage, error) {
	if len(meta) == 0 {
		return json.RawMessage("{}"), nil
	}
	return json.Marshal(meta)
}
