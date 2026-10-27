package keyleak

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *handler) disable(ctx context.Context, key db.FindLeakedKeysByHashesRow, token reportedToken) error {
	return db.TxRetry(ctx, h.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		legacyChanged, err := queries.DisableLeakedKey(ctx, db.DisableLeakedKeyParams{
			Hash:        key.Hash,
			WorkspaceID: sql.NullString{String: key.WorkspaceID, Valid: true},
			Now:         sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
		if err != nil {
			return err
		}
		rootChanged, err := queries.DisableLeakedRootKey(ctx, db.DisableLeakedRootKeyParams{
			Hash: key.Hash, WorkspaceID: key.WorkspaceID,
		})
		if err != nil {
			return err
		}
		if legacyChanged+rootChanged == 0 {
			return nil
		}

		event := auditlog.KeyLeakedEvent
		if key.KeyType == "rootKey" {
			event = auditlog.RootKeyLeakedEvent
		}
		return h.auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   key.WorkspaceID,
			Event:         event,
			Display:       "Automatically disabled leaked key after GitHub secret scanning detected exposure",
			ActorType:     auditlog.SystemActor,
			ActorID:       "github_secret_scanning",
			ActorName:     "GitHub secret scanning",
			ActorMeta:     nil,
			RemoteIP:      "",
			UserAgent:     "",
			CorrelationID: "",
			Targets: []auditlog.AuditLogTarget{{
				Type:        auditlog.KeyTargetType,
				ID:          key.ID,
				DisplayName: "",
				Name:        "",
				Meta:        map[string]any{"source": token.Source, "url": token.URL, "enabled": false},
			}},
		}})
	})
}
