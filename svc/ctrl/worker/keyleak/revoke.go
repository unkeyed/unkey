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

// revoke reports whether this invocation deleted the key. The deletion time is
// journaled before the transaction, so a retry after an attempt that committed
// but was not journaled still recognizes the deletion as its own.
func (s *Service) revoke(ctx restate.Context, req *hydrav1.KeyLeakNotification) (bool, error) {
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
	deletedAt := sql.NullInt64{Int64: now.UnixMilli(), Valid: true}
	display := fmt.Sprintf("Revoked %s %s after %s reported it publicly exposed", store.noun, req.GetKeyId(), reporterOf(req).name)
	return restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
		return db.TxWithResultRetry(rc, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
			queries := db.NewQueries(tx)
			rows, err := store.revoke(txCtx, queries, req.GetKeyId(), req.GetKeyHash(), deletedAt)
			if err != nil {
				return false, err
			}
			if rows == 0 {
				current, err := store.find(txCtx, queries, req.GetKeyId())
				if db.IsNotFound(err) {
					return false, nil
				}
				return current.hash == req.GetKeyHash() && current.deletedAt == deletedAt, err
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
	}, runOptions("revoke leaked key")...)
}

type keyStore struct {
	noun   string
	event  auditlog.AuditLogEvent
	revoke func(ctx context.Context, queries *db.Queries, keyID, keyHash string, deletedAt sql.NullInt64) (int64, error)
	find   func(ctx context.Context, queries *db.Queries, keyID string) (storedKey, error)
}

type storedKey struct {
	hash      string
	deletedAt sql.NullInt64
}

var keyStores = map[hydrav1.KeyLeakKeyStore]keyStore{
	hydrav1.KeyLeakKeyStore_KEY_LEAK_KEY_STORE_ROOT_KEYS: {
		noun:  "root key",
		event: auditlog.RootKeyDeleteEvent,
		revoke: func(ctx context.Context, queries *db.Queries, keyID, keyHash string, deletedAt sql.NullInt64) (int64, error) {
			return queries.RevokeLeakedUnkeyRootKey(ctx, db.RevokeLeakedUnkeyRootKeyParams{Now: deletedAt, ID: keyID, Hash: keyHash})
		},
		find: func(ctx context.Context, queries *db.Queries, keyID string) (storedKey, error) {
			key, err := queries.FindUnkeyRootKeyByID(ctx, keyID)
			return storedKey{hash: key.Hash, deletedAt: key.DeletedAt}, err
		},
	},
	hydrav1.KeyLeakKeyStore_KEY_LEAK_KEY_STORE_KEYS: {
		noun:  "key",
		event: auditlog.KeyDeleteEvent,
		revoke: func(ctx context.Context, queries *db.Queries, keyID, keyHash string, deletedAt sql.NullInt64) (int64, error) {
			return queries.RevokeLeakedKey(ctx, db.RevokeLeakedKeyParams{Now: deletedAt, ID: keyID, Hash: keyHash})
		},
		find: func(ctx context.Context, queries *db.Queries, keyID string) (storedKey, error) {
			key, err := queries.FindKeyByID(ctx, keyID)
			return storedKey{hash: key.Hash, deletedAt: key.DeletedAtM}, err
		},
	},
}
