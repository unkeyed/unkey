package auditlogs

import (
	"context"
	"database/sql"
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

// Insert implements AuditLogService.Insert, persisting audit logs and their
// associated resource targets to the database within a transactional context.
// Insert handles batch processing of multiple audit logs, automatically managing
// transaction lifecycle, ID generation, and metadata serialization.
//
// During the dual-write window each call writes to two surfaces in the same
// MySQL transaction:
//   - The legacy `audit_log` and `audit_log_target` tables, which the
//     dashboard still reads from. This keeps the existing read path working
//     and gives us a clean rollback (revert the deploy and the system runs
//     unchanged).
//   - The new `clickhouse_outbox` table, drained to ClickHouse by the
//     AuditLogExportService worker. After the dashboard cuts over to CH and
//     the historical backfill completes, the legacy writes can be removed.
//
// When tx is nil, Insert creates its own transaction so the legacy + outbox
// writes commit atomically.
//
// The method handles several important edge cases:
//   - Empty log slices return immediately without database interaction
//   - Missing bucket names are automatically set to DefaultBucket
//   - JSON serialization failures for metadata cause immediate error return
//   - Transaction rollback is handled gracefully with error logging
//   - Context cancellation triggers automatic cleanup
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
	auditLogs := make([]db.InsertAuditLogParams, 0, len(logs))
	auditLogTargets := make([]db.InsertAuditLogTargetParams, 0)
	outboxRows := make([]db.InsertClickhouseOutboxParams, 0, len(logs))

	for _, l := range logs {
		auditLogID := uid.New(uid.AuditLogPrefix)
		now := time.Now().UnixMilli()

		actorMetaJSON, err := json.Marshal(l.ActorMeta)
		if err != nil {
			return err
		}

		auditLogs = append(auditLogs, db.InsertAuditLogParams{
			ID:          auditLogID,
			WorkspaceID: l.WorkspaceID,
			BucketID:    "dummy",
			Bucket:      DefaultBucket,
			Event:       string(l.Event),
			Display:     l.Display,
			ActorMeta:   actorMetaJSON,
			ActorType:   string(l.ActorType),
			ActorID:     l.ActorID,
			ActorName:   sql.NullString{String: l.ActorName, Valid: l.ActorName != ""},
			RemoteIp:    sql.NullString{String: l.RemoteIP, Valid: l.RemoteIP != ""},
			UserAgent:   sql.NullString{String: l.UserAgent, Valid: l.UserAgent != ""},
			Time:        now,
			CreatedAt:   now,
		})

		targets := make([]auditlog.EventTarget, 0, len(l.Resources))
		for _, resource := range l.Resources {
			targetMetaJSON, err := json.Marshal(resource.Meta)
			if err != nil {
				return err
			}

			auditLogTargets = append(auditLogTargets, db.InsertAuditLogTargetParams{
				ID:          resource.ID,
				AuditLogID:  auditLogID,
				WorkspaceID: l.WorkspaceID,
				BucketID:    "dummy",
				Bucket:      DefaultBucket,
				Type:        string(resource.Type),
				DisplayName: resource.DisplayName,
				Name:        sql.NullString{String: resource.DisplayName, Valid: resource.DisplayName != ""},
				Meta:        targetMetaJSON,
				CreatedAt:   now,
			})

			targets = append(targets, auditlog.EventTarget{
				Type: string(resource.Type),
				ID:   resource.ID,
				Name: resource.DisplayName,
				Meta: resource.Meta,
			})
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
			RemoteIP:  l.RemoteIP,
			UserAgent: l.UserAgent,
			Meta:      nil,
			Targets:   targets,
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

	if err := db.BulkQuery.InsertAuditLogs(ctx, tx, auditLogs); err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to insert audit logs"), fault.Public("Failed to insert audit logs"),
		)
	}

	if err := db.BulkQuery.InsertAuditLogTargets(ctx, tx, auditLogTargets); err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to insert audit log targets"), fault.Public("Failed to insert audit log targets"),
		)
	}

	if err := db.BulkQuery.InsertClickhouseOutboxes(ctx, tx, outboxRows); err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to insert clickhouse outbox rows"), fault.Public("Failed to insert audit logs"),
		)
	}

	return nil
}
