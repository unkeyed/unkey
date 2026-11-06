package keyleak

import (
	"context"
	"database/sql"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// disable reports whether the reported key is disabled and still matches the
// report once it returns. A key that was already disabled counts, so a retry
// after an attempt that committed but was not journaled still alerts, and a
// workspace that had disabled the key itself still learns that it leaked. Only
// the attempt that flips the key writes an audit log.
func (s *Service) disable(ctx restate.Context, req *hydrav1.KeyLeakNotification) (bool, error) {
	store, found := keyStores[req.GetKeyStore()]
	if err := assert.All(
		assert.True(found, "key store must be known"),
		assert.NotEmpty(req.GetKeyId(), "key id is required"),
		assert.NotEmpty(req.GetWorkspaceId(), "workspace id is required"),
		assert.NotEmpty(req.GetKeyHash(), "key hash is required"),
	); err != nil {
		return false, restate.ToTerminalError(err)
	}

	now, err := restateutil.Now(ctx)
	if err != nil {
		return false, err
	}
	display := fmt.Sprintf("Disabled %s %s after %s reported it publicly exposed", store.noun, req.GetKeyId(), reporterOf(req).name)
	return restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
		return db.TxWithResultRetry(rc, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
			queries := db.NewQueries(tx)
			rows, err := store.disable(txCtx, queries, req.GetKeyId(), req.GetKeyHash(), now.UnixMilli())
			if err != nil {
				return false, err
			}
			if rows == 0 {
				current, err := store.find(txCtx, queries, req.GetKeyId())
				if db.IsNotFound(err) {
					return false, nil
				}
				return current.hash == req.GetKeyHash() && !current.deletedAt.Valid && !current.enabled, err
			}
			if err := assert.Equal(rows, int64(1), "key ids are unique"); err != nil {
				return false, err
			}
			return true, s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{{
				Event:         store.event,
				WorkspaceID:   req.GetWorkspaceId(),
				Display:       display,
				ActorID:       "",
				ActorType:     auditlog.SystemActor,
				ActorName:     "",
				ActorMeta:     map[string]any{},
				RemoteIP:      "",
				UserAgent:     "",
				CorrelationID: "",
				Targets: []auditlog.AuditLogTarget{{
					Type: auditlog.KeyTargetType, ID: req.GetKeyId(), Name: req.GetKeyId(), DisplayName: req.GetKeyId(), Meta: map[string]any{},
				}},
			}})
		})
	}, runOptions("disable leaked key")...)
}

type keyStore struct {
	noun    string
	event   auditlog.AuditLogEvent
	disable func(ctx context.Context, queries *db.Queries, keyID, keyHash string, nowMs int64) (int64, error)
	find    func(ctx context.Context, queries *db.Queries, keyID string) (storedKey, error)
}

type storedKey struct {
	hash      string
	enabled   bool
	deletedAt sql.NullInt64
}

var keyStores = map[hydrav1.KeyLeakKeyStore]keyStore{
	hydrav1.KeyLeakKeyStore_KEY_LEAK_KEY_STORE_ROOT_KEYS: {
		noun:  "root key",
		event: auditlog.RootKeyUpdateEvent,
		disable: func(ctx context.Context, queries *db.Queries, keyID, keyHash string, _ int64) (int64, error) {
			return queries.DisableLeakedUnkeyRootKey(ctx, db.DisableLeakedUnkeyRootKeyParams{ID: keyID, Hash: keyHash})
		},
		find: func(ctx context.Context, queries *db.Queries, keyID string) (storedKey, error) {
			key, err := queries.FindUnkeyRootKeyByID(ctx, keyID)
			return storedKey{hash: key.Hash, enabled: key.Enabled, deletedAt: key.DeletedAt}, err
		},
	},
	hydrav1.KeyLeakKeyStore_KEY_LEAK_KEY_STORE_KEYS: {
		noun:  "key",
		event: auditlog.KeyUpdateEvent,
		disable: func(ctx context.Context, queries *db.Queries, keyID, keyHash string, nowMs int64) (int64, error) {
			return queries.DisableLeakedKey(ctx, db.DisableLeakedKeyParams{Now: sql.NullInt64{Int64: nowMs, Valid: true}, ID: keyID, Hash: keyHash})
		},
		find: func(ctx context.Context, queries *db.Queries, keyID string) (storedKey, error) {
			key, err := queries.FindKeyByID(ctx, keyID)
			return storedKey{hash: key.Hash, enabled: key.Enabled, deletedAt: key.DeletedAtM}, err
		},
	},
}
