package auditlogs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *service) Insert(ctx context.Context, tx *sql.Tx, logs []auditlog.AuditLog) error {
	auditLogs := make([]db.InsertAuditLogParams, 0)
	auditLogTargets := make([]db.InsertAuditLogTargetParams, 0)

	if len(logs) == 0 {
		return nil
	}

	var dbTx db.DBTX = tx
	if tx == nil {
		dbTx = s.db.RW()
	}

	for _, l := range logs {
		auditLogID := uid.New(uid.AuditLogPrefix)
		now := time.Now().UnixMilli()

		if l.Bucket == "" {
			l.Bucket = "unkey_mutations"
		}

		cacheKey := fmt.Sprintf("%s:%s", l.WorkspaceID, l.Bucket)
		bucketID, err := s.bucketCache.SWR(
			ctx,
			cacheKey,
			func(ctx context.Context) (string, error) {
				return db.Query.FindAuditLogBucketIDByWorkspaceIDAndName(ctx, s.db.RO(), db.FindAuditLogBucketIDByWorkspaceIDAndNameParams{
					WorkspaceID: l.WorkspaceID,
					Name:        l.Bucket,
				})
			},
			func(err error) cache.CacheHit {
				if errors.Is(err, sql.ErrNoRows) {
					return cache.Null
				}

				return cache.Miss
			},
		)

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error("Failed to fetch audit log bucket", "workspaceID", l.WorkspaceID, "bucket", l.Bucket, "error", err)
			continue
		}

		if errors.Is(err, sql.ErrNoRows) {
			bucketID = uid.New(uid.AuditLogBucketPrefix)
			err = db.Query.InsertAuditLogBucket(ctx, s.db.RW(), db.InsertAuditLogBucketParams{
				ID:            bucketID,
				WorkspaceID:   l.WorkspaceID,
				Name:          l.Bucket,
				CreatedAt:     time.Now().UnixMilli(),
				RetentionDays: sql.NullInt32{Int32: 90, Valid: true},
			})
			if err != nil {
				s.logger.Error("Failed to insert audit log bucket", "workspaceID", l.WorkspaceID, "bucket", l.Bucket, "error", err)
				continue
			}

			s.bucketCache.Set(ctx, cacheKey, bucketID)
		}

		auditLogs = append(auditLogs, db.InsertAuditLogParams{
			ID:          auditLogID,
			WorkspaceID: l.WorkspaceID,
			BucketID:    bucketID,
			Event:       string(l.Event),
			Display:     l.Display,
			ActorMeta:   l.Actor.Meta,
			ActorType:   string(l.Actor.Type),
			ActorID:     l.Actor.ID,
			ActorName:   sql.NullString{String: l.Actor.Name, Valid: l.Actor.Name != ""},
			RemoteIp:    sql.NullString{String: l.RemoteIP, Valid: l.RemoteIP != ""},
			UserAgent:   sql.NullString{String: l.UserAgent, Valid: l.UserAgent != ""},
			Time:        now,
			CreatedAt:   now,
		})

		for _, resource := range l.Resources {
			auditLogTargets = append(auditLogTargets, db.InsertAuditLogTargetParams{
				ID:          resource.ID,
				AuditLogID:  auditLogID,
				WorkspaceID: l.WorkspaceID,
				BucketID:    bucketID,
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
