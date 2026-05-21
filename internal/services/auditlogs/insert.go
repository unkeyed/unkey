package auditlogs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// DefaultBucket is the default bucket name used for audit logs when no bucket
// is specified. All audit logs are categorized into buckets for organization
// and querying purposes, with "unkey_mutations" serving as the standard bucket
// for most operational audit events.
const (
	DefaultBucket = "unkey_mutations"
)

// Insert implements AuditLogService.Insert, persisting audit logs to the
// `clickhouse_outbox` MySQL table within a transactional context. The
// AuditLogExportService worker drains the outbox and ships each row to
// ClickHouse `audit_logs_raw_v1`.
//
// When tx is nil, Insert opens its own transaction so the outbox INSERT
// commits atomically with whatever the caller is doing. Caller-provided
// tx is reused so the outbox row commits with the underlying mutation
// (the outbox pattern's whole point — durability of the mutation equals
// durability of the audit row).
//
// Edge cases:
//   - Empty log slices return immediately without touching the database.
//   - JSON serialization failures for the envelope (actor / targets /
//     meta) cause immediate error return.
func (s *service) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	if tx == nil {
		return db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
			return s.insertLogs(ctx, tx, logs)
		})
	}

	return s.insertLogs(ctx, tx, logs)
}

func (s *service) insertLogs(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	outboxRows := make([]db.InsertClickhouseOutboxParams, 0, len(logs))

	// Resolve a shared correlation ID for the batch. Precedence:
	//   1. ctx-scoped value via auditlog.WithCorrelation (multi-Insert flows)
	//   2. Auto-mint when len(logs) > 1 (batched single-Insert flows)
	//   3. Empty (single-event flows that don't need grouping)
	//
	// Per-event CorrelationID set on the struct still wins over this; the
	// shared default only fills in when the caller didn't set one.
	sharedCorrelationID := auditlog.CorrelationFrom(ctx)
	if sharedCorrelationID == "" && len(logs) > 1 {
		sharedCorrelationID = auditlog.NewCorrelationID()
	}

	for _, l := range logs {
		auditLogID := uid.New(uid.AuditLogPrefix)
		now := time.Now().UnixMilli()

		targets := make([]auditlog.EventTarget, 0, len(l.Resources))
		for _, resource := range l.Resources {
			targets = append(targets, auditlog.EventTarget{
				Type: string(resource.Type),
				ID:   resource.ID,
				Name: resource.DisplayName,
				Meta: resource.Meta,
			})
		}

		correlationID := l.CorrelationID
		if correlationID == "" {
			correlationID = sharedCorrelationID
		}

		envelope := auditlog.Event{
			EventID:     auditLogID,
			Time:        now,
			WorkspaceID: l.WorkspaceID,
			Bucket:      DefaultBucket,
			Source:      auditlog.EventSourcePlatform,
			Event:       string(l.Event),
			Description: l.Display,
			Actor: auditlog.EventActor{
				Type: string(l.ActorType),
				ID:   l.ActorID,
				Name: l.ActorName,
				Meta: l.ActorMeta,
			},
			RemoteIP:      l.RemoteIP,
			UserAgent:     l.UserAgent,
			Meta:          nil,
			Targets:       targets,
			CorrelationID: correlationID,
		}
		payload, err := json.Marshal(envelope)
		if err != nil {
			return err
		}

		outboxRows = append(outboxRows, db.InsertClickhouseOutboxParams{
			Version:     auditlog.OutboxVersionV1,
			WorkspaceID: l.WorkspaceID,
			EventID:     auditLogID,
			Payload:     payload,
			CreatedAt:   now,
		})
	}

	if err := db.BulkQuery.InsertClickhouseOutboxes(ctx, tx, outboxRows); err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to insert clickhouse outbox rows"), fault.Public("Failed to insert audit logs"),
		)
	}

	return nil
}
