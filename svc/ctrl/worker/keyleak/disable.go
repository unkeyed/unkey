package keyleak

import (
	"context"
	"database/sql"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) disableBatch(ctx context.Context, req *hydrav1.KeyLeakBatchRequest) error {
	hashes := make([]string, 0, len(req.GetKeys()))
	byHash := make(map[string]*hydrav1.KeyLeakMatch, len(req.GetKeys()))
	for _, key := range req.GetKeys() {
		hashes = append(hashes, key.GetHash())
		byHash[key.GetHash()] = key
	}
	return db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		legacy, err := queries.FindEnabledKeysByHashesForUpdate(ctx, hashes)
		if err != nil {
			return err
		}
		roots, err := queries.FindEnabledRootKeysByHashesForUpdate(ctx, hashes)
		if err != nil {
			return err
		}

		changed := make(map[string]bool)
		legacyIDs := make([]string, 0, len(legacy))
		for _, key := range legacy {
			workspaceID := key.WorkspaceID
			if key.ForWorkspaceID.Valid {
				workspaceID = key.ForWorkspaceID.String
			}
			if byHash[key.Hash].GetWorkspaceId() == workspaceID {
				legacyIDs = append(legacyIDs, key.ID)
				changed[key.Hash] = true
			}
		}
		rootIDs := make([]string, 0, len(roots))
		for _, key := range roots {
			if byHash[key.Hash].GetWorkspaceId() == key.WorkspaceID {
				rootIDs = append(rootIDs, key.ID)
				changed[key.Hash] = true
			}
		}
		if len(changed) == 0 {
			return nil
		}

		if err := queries.DisableLeakedKeys(ctx, db.DisableLeakedKeysParams{
			Ids: legacyIDs, Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		}); err != nil {
			return err
		}
		if err := queries.DisableLeakedRootKeys(ctx, rootIDs); err != nil {
			return err
		}
		logs := make([]auditlog.AuditLog, 0, len(changed))
		for _, key := range req.GetKeys() {
			if !changed[key.GetHash()] {
				continue
			}
			delete(changed, key.GetHash())
			event := auditlog.KeyLeakedEvent
			if key.GetRootKey() {
				event = auditlog.RootKeyLeakedEvent
			}
			logs = append(logs, auditlog.AuditLog{
				WorkspaceID: key.GetWorkspaceId(), Event: event,
				Display:       "Automatically disabled leaked key after GitHub secret scanning detected exposure",
				ActorType:     auditlog.SystemActor,
				ActorID:       "github_secret_scanning",
				ActorName:     "GitHub secret scanning",
				ActorMeta:     map[string]any{"source": "github"},
				RemoteIP:      "",
				UserAgent:     "",
				CorrelationID: req.GetReportId(),
				Targets: []auditlog.AuditLogTarget{{
					Type: auditlog.KeyTargetType, ID: key.GetNotification().GetKeyId(), DisplayName: "", Name: "",
					Meta: map[string]any{"source": key.GetNotification().GetSource(), "url": key.GetNotification().GetUrl(), "enabled": false},
				}},
			})
		}
		return s.auditlogs.Insert(ctx, tx, logs)
	})
}
