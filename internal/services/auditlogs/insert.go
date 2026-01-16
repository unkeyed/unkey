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
// The method creates two types of database records for each audit log:
//   - Primary audit log records containing event details and actor information
//   - Resource target records linking the audit event to affected resources
//
// When tx is nil, Insert creates its own transaction to ensure all logs are
// committed atomically. This prevents partial audit log insertion that could
// result in compliance gaps or inconsistent audit trails.
//
// The method handles several important edge cases:
//   - Empty log slices return immediately without database interaction
//   - Missing bucket names are automatically set to DefaultBucket
//   - JSON serialization failures for metadata cause immediate error return
//   - Transaction rollback is handled gracefully with error logging
//   - Context cancellation triggers automatic cleanup
//
// All audit logs receive unique identifiers and consistent timestamps to
// maintain proper audit trail ordering and prevent ID conflicts in
// high-concurrency scenarios.
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
	auditLogs := make([]db.InsertAuditLogParams, 0)
	auditLogTargets := make([]db.InsertAuditLogTargetParams, 0)

	for _, l := range logs {
		auditLogID := uid.New(uid.AuditLogPrefix)

		now := time.Now().UnixMilli()
		actorMeta, err := json.Marshal(l.ActorMeta)
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
			ActorMeta:   actorMeta,
			ActorType:   string(l.ActorType),
			ActorID:     l.ActorID,
			ActorName:   sql.NullString{String: l.ActorName, Valid: l.ActorName != ""},
			RemoteIp:    sql.NullString{String: l.RemoteIP, Valid: l.RemoteIP != ""},
			UserAgent:   sql.NullString{String: l.UserAgent, Valid: l.UserAgent != ""},
			Time:        now,
			CreatedAt:   now,
		})

		for _, resource := range l.Resources {
			meta, err := json.Marshal(resource.Meta)
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
				Meta:        meta,
				CreatedAt:   now,
			})
		}
	}

	err := db.BulkQuery.InsertAuditLogs(ctx, tx, auditLogs)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to insert audit logs"), fault.Public("Failed to insert audit logs"),
		)
	}

	err = db.BulkQuery.InsertAuditLogTargets(ctx, tx, auditLogTargets)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to insert audit log targets"), fault.Public("Failed to insert audit log targets"),
		)
	}

	return nil
}
