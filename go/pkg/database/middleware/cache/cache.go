package cache

import (
	"context"
	"errors"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type cacheMiddleware struct {
	db database.Database

	translateError func(error) cache.CacheHit

	keyByHash                      cache.Cache[string, entities.Key]
	workspaceByID                  cache.Cache[string, entities.Workspace]
	keyByID                        cache.Cache[string, entities.Key]
	ratelimitNamespaceByID         cache.Cache[string, entities.RatelimitNamespace]
	ratelimitNamespaceByName       cache.Cache[KeyRatelimitNamespaceByName, entities.RatelimitNamespace]
	ratelimitOverridesByIdentifier cache.Cache[KeyRatelimitOverridesByIdentifier, []entities.RatelimitOverride]
	ratelimitOverrideByID          cache.Cache[KeyRatelimitOverrideByID, entities.RatelimitOverride]
	permissionsByKeyID             cache.Cache[string, []string]
}

var _ database.Database = (*cacheMiddleware)(nil)

func WithCaching(logger logging.Logger) database.Middleware {

	clk := clock.New()
	return func(db database.Database) database.Database {

		return &cacheMiddleware{
			db: db,
			translateError: func(err error) cache.CacheHit {
				if err == nil {
					return cache.Hit
				}
				if errors.Is(err, database.ErrNotFound) {
					// if no data was found, we store a special NullEntry
					return cache.Null
				}
				// some other error, which we don't want to cache
				return cache.Miss

			},
			keyByHash: cache.New[string, entities.Key](cache.Config[string, entities.Key]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "key_by_hash",
				Clock:    clk,
			}),
			keyByID: cache.New[string, entities.Key](cache.Config[string, entities.Key]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "key_by_id",
				Clock:    clk,
			}),
			workspaceByID: cache.New[string, entities.Workspace](cache.Config[string, entities.Workspace]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "workspace_by_id",
				Clock:    clk,
			}),
			ratelimitNamespaceByID: cache.New[string, entities.RatelimitNamespace](cache.Config[string, entities.RatelimitNamespace]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "ratelimit_namespace_by_id",
				Clock:    clk,
			}),
			ratelimitNamespaceByName: cache.New[KeyRatelimitNamespaceByName, entities.RatelimitNamespace](cache.Config[KeyRatelimitNamespaceByName, entities.RatelimitNamespace]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "ratelimit_namespace_by_name",
				Clock:    clk,
			}),
			ratelimitOverridesByIdentifier: cache.New[KeyRatelimitOverridesByIdentifier, []entities.RatelimitOverride](cache.Config[KeyRatelimitOverridesByIdentifier, []entities.RatelimitOverride]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "ratelimit_overrides_by_identifier",
				Clock:    clk,
			}),
			ratelimitOverrideByID: cache.New[KeyRatelimitOverrideByID, entities.RatelimitOverride](cache.Config[KeyRatelimitOverrideByID, entities.RatelimitOverride]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "ratelimit_override_by_id",
				Clock:    clk,
			}),
			permissionsByKeyID: cache.New[string, []string](cache.Config[string, []string]{
				Fresh:    10 * time.Second,
				Stale:    1 * time.Minute,
				Logger:   logger,
				MaxSize:  1_000_000,
				Resource: "permissions_by_key_id",
				Clock:    clk,
			}),
		}
	}
}

func (c *cacheMiddleware) InsertWorkspace(ctx context.Context, workspace entities.Workspace) error {
	return c.db.InsertWorkspace(ctx, workspace)
}
func (c *cacheMiddleware) FindWorkspaceByID(ctx context.Context, id string) (entities.Workspace, error) {
	return c.workspaceByID.SWR(ctx, id, func(refreshCtx context.Context) (entities.Workspace, error) {
		return c.db.FindWorkspaceByID(refreshCtx, id)
	}, c.translateError)

}
func (c *cacheMiddleware) UpdateWorkspacePlan(ctx context.Context, workspaceID string, plan entities.WorkspacePlan) error {
	return c.db.UpdateWorkspacePlan(ctx, workspaceID, plan)
}
func (c *cacheMiddleware) UpdateWorkspaceEnabled(ctx context.Context, id string, enabled bool) error {
	return c.db.UpdateWorkspaceEnabled(ctx, id, enabled)
}
func (c *cacheMiddleware) DeleteWorkspace(ctx context.Context, id string, hardDelete bool) error {
	return c.db.DeleteWorkspace(ctx, id, hardDelete)
}

func (c *cacheMiddleware) InsertKeyring(ctx context.Context, keyring entities.Keyring) error {
	return c.db.InsertKeyring(ctx, keyring)
}

func (c *cacheMiddleware) InsertKey(ctx context.Context, key entities.Key) error {
	return c.db.InsertKey(ctx, key)
}
func (c *cacheMiddleware) FindKeyByID(ctx context.Context, keyID string) (key entities.Key, err error) {
	return c.keyByID.SWR(ctx, keyID, func(refreshCtx context.Context) (entities.Key, error) {
		return c.db.FindKeyByID(refreshCtx, keyID)
	}, c.translateError)
}
func (c *cacheMiddleware) FindKeyByHash(ctx context.Context, hash string) (key entities.Key, err error) {
	return c.keyByHash.SWR(ctx, hash, func(refreshCtx context.Context) (entities.Key, error) {
		return c.db.FindKeyByHash(refreshCtx, hash)
	}, c.translateError)
}
func (c *cacheMiddleware) FindPermissionsByKeyID(ctx context.Context, keyID string) (permissions []string, err error) {
	return c.permissionsByKeyID.SWR(ctx, keyID, func(refreshCtx context.Context) ([]string, error) {
		return c.db.FindPermissionsByKeyID(refreshCtx, keyID)
	}, c.translateError)
}
func (c *cacheMiddleware) FindKeyForVerification(ctx context.Context, hash string) (key entities.Key, err error) {
	panic("IMPLEMENT ME")
}
func (c *cacheMiddleware) InsertRatelimitNamespace(ctx context.Context, namespace entities.RatelimitNamespace) error {
	return c.db.InsertRatelimitNamespace(ctx, namespace)
}
func (c *cacheMiddleware) FindRatelimitNamespaceByID(ctx context.Context, namespaceID string) (entities.RatelimitNamespace, error) {
	return c.ratelimitNamespaceByID.SWR(ctx, namespaceID, func(refreshCtx context.Context) (entities.RatelimitNamespace, error) {
		return c.db.FindRatelimitNamespaceByID(refreshCtx, namespaceID)
	}, c.translateError)
}
func (c *cacheMiddleware) FindRatelimitNamespaceByName(ctx context.Context, workspaceID string, name string) (entities.RatelimitNamespace, error) {
	return c.ratelimitNamespaceByName.SWR(ctx, KeyRatelimitNamespaceByName{WorkspaceID: workspaceID, NamespaceName: name}, func(refreshCtx context.Context) (entities.RatelimitNamespace, error) {
		return c.db.FindRatelimitNamespaceByName(refreshCtx, workspaceID, name)
	}, c.translateError)
}
func (c *cacheMiddleware) DeleteRatelimitNamespace(ctx context.Context, id string) error {
	return c.db.DeleteRatelimitNamespace(ctx, id)
}
func (c *cacheMiddleware) InsertRatelimitOverride(ctx context.Context, ratelimitOverride entities.RatelimitOverride) error {
	return c.db.InsertRatelimitOverride(ctx, ratelimitOverride)
}
func (c *cacheMiddleware) FindRatelimitOverridesByIdentifier(ctx context.Context, workspaceID, namespaceID, identifier string) (ratelimitOverrides []entities.RatelimitOverride, err error) {
	return c.ratelimitOverridesByIdentifier.SWR(ctx, KeyRatelimitOverridesByIdentifier{WorkspaceID: workspaceID, NamespaceID: namespaceID, Identifier: identifier}, func(refreshCtx context.Context) ([]entities.RatelimitOverride, error) {
		return c.db.FindRatelimitOverridesByIdentifier(refreshCtx, workspaceID, namespaceID, identifier)
	}, c.translateError)
}
func (c *cacheMiddleware) FindRatelimitOverrideByID(ctx context.Context, workspaceID, overrideID string) (ratelimitOverrides entities.RatelimitOverride, err error) {
	return c.ratelimitOverrideByID.SWR(ctx, KeyRatelimitOverrideByID{WorkspaceID: workspaceID, OverrideID: overrideID}, func(refreshCtx context.Context) (entities.RatelimitOverride, error) {
		return c.db.FindRatelimitOverrideByID(refreshCtx, workspaceID, overrideID)
	}, c.translateError)
}
func (c *cacheMiddleware) UpdateRatelimitOverride(ctx context.Context, override entities.RatelimitOverride) error {
	return c.db.UpdateRatelimitOverride(ctx, override)
}
func (c *cacheMiddleware) DeleteRatelimitOverride(ctx context.Context, id string) error {
	return c.db.DeleteRatelimitOverride(ctx, id)
}
func (c *cacheMiddleware) Close() error {
	return c.db.Close()
}
