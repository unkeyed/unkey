package auditlogs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// DEFAULT_BUCKET is the default bucket name used for audit logs when no bucket
// is specified. All audit logs are categorized into buckets for organization
// and querying purposes, with "unkey_mutations" serving as the standard bucket
// for most operational audit events.
const (
	DEFAULT_BUCKET = "unkey_mutations"
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
//   - Missing bucket names are automatically set to DEFAULT_BUCKET
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

	auditLogs := make([]db.InsertAuditLogParams, 0)
	auditLogTargets := make([]db.InsertAuditLogTargetParams, 0)

	var dbTx = tx
	var rwTx *sql.Tx
	if tx == nil {
		// If we didn't get a transaction, start a new one so we can commit all
		// audit logs together to not miss anything
		newTx, err := s.db.RW().Begin(ctx)
		if err != nil {
			return err
		}

		dbTx = newTx
		rwTx = newTx

		defer func() {
			rollbackErr := rwTx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				s.logger.Error("rollback failed", "error", rollbackErr)
			}
		}()
	}

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
			Bucket:      DEFAULT_BUCKET,
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
				Bucket:      DEFAULT_BUCKET,

				Type:        string(resource.Type),
				DisplayName: resource.DisplayName,
				Name:        sql.NullString{String: resource.DisplayName, Valid: resource.DisplayName != ""},
				Meta:        meta,
				CreatedAt:   now,
			})
		}
	}

	for _, log := range auditLogs {
		if err := db.Query.InsertAuditLog(ctx, dbTx, log); err != nil {
			return err
		}
	}

	for _, logTarget := range auditLogTargets {
		if err := db.Query.InsertAuditLogTarget(ctx, dbTx, logTarget); err != nil {
			return err
		}
	}

	// If we are not using a transaction that has been passed in we will just commit all logs
	if rwTx != nil {
		if err := rwTx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
