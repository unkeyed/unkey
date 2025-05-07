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

const (
	// DEFAULT_BUCKET is the default bucket name used for audit logs
	// when no specific bucket is provided.
	DEFAULT_BUCKET = "unkey_mutations"
)

// Insert records a batch of audit logs to the database, optionally as part of an
// existing transaction. If tx is nil, a new transaction is created.
//
// This method transforms the provided audit logs into the database schema format,
// then inserts them and their associated targets atomically. If a transaction is
// provided, the logs will be part of that transaction, allowing audit logs to be
// committed alongside other database changes.
//
// The method ensures all logs have a valid bucket name (defaulting to DEFAULT_BUCKET),
// and generates unique IDs for each log entry. It also handles the conversion of
// metadata fields to JSON for storage.
//
// If a new transaction is created by this method, it will be committed if all
// inserts succeed, and rolled back if any error occurs.
//
// This method is commonly used when making important changes to system state,
// such as key creation, updates, or deletions, to maintain an audit trail.
//
// Parameters:
//   - ctx: Context for the operation, allowing for cancellation and timeout
//   - tx: Optional existing transaction to use. If nil, a new one is created
//   - logs: The audit logs to insert
//
// Returns an error if any part of the insertion process fails.
func (s *service) Insert(ctx context.Context, tx *sql.Tx, logs []auditlog.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	auditLogs := make([]db.InsertAuditLogParams, 0)
	auditLogTargets := make([]db.InsertAuditLogTargetParams, 0)

	var dbTx db.DBTX = tx
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
		if l.Bucket == "" {
			l.Bucket = DEFAULT_BUCKET
		}

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
