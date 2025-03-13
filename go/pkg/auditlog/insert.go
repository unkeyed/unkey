package auditlog

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type AuditLog struct {
	Event AuditLogEvent

	WorkspaceID string
	Display     string

	Actor AuditLogActorData

	Resources []AuditLogResource
}

type AuditLogActorData struct {
	ID   string
	Type AuditLogActor
	Name string
	Meta []byte
}

type AuditLogResource struct {
	ID          string
	Name        string
	DisplayName string
	Meta        []byte
	Type        AuditLogResourceType
}

type Services struct {
	DB db.Database
}

// InsertAuditLog records multiple audit log entries in a given transaction.
// It will start a new transaction if none is provided.
func InsertAuditLog(ctx context.Context, svc Services, tx *sql.Tx, s *zen.Session, logs []AuditLog) error {
	auditLogs := make([]db.InsertAuditLogParams, 0)
	auditLogTargets := make([]db.InsertAuditLogTargetParams, 0)

	if len(logs) == 0 {
		return nil
	}

	var dbTx db.DBTX = tx
	if tx == nil {
		dbTx = svc.DB.RW()
	}

	for _, log := range logs {
		auditLogID := uid.New(uid.AuditLogPrefix)
		now := time.Now().UnixMilli()

		bucketId := ""

		auditLogs = append(auditLogs, db.InsertAuditLogParams{
			ID:          auditLogID,
			WorkspaceID: log.WorkspaceID,
			BucketID:    bucketId,
			Event:       string(log.Event),
			Display:     log.Display,
			ActorMeta:   log.Actor.Meta,
			ActorType:   string(log.Actor.Type),
			ActorID:     log.Actor.ID,
			ActorName:   sql.NullString{String: log.Actor.Name, Valid: log.Actor.Name != ""},
			RemoteIp:    sql.NullString{String: s.Location(), Valid: s.Location() != ""},
			UserAgent:   sql.NullString{String: s.UserAgent(), Valid: s.UserAgent() != ""},
			Time:        now,
			CreatedAt:   now,
		})

		for _, resource := range log.Resources {
			auditLogTargets = append(auditLogTargets, db.InsertAuditLogTargetParams{
				ID:          resource.ID,
				AuditLogID:  auditLogID,
				WorkspaceID: log.WorkspaceID,
				BucketID:    bucketId,
				Type:        string(resource.Type),
				DisplayName: resource.DisplayName,
				Name:        sql.NullString{String: resource.DisplayName, Valid: resource.DisplayName != ""},
				Meta:        resource.Meta,
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

	return nil
}
