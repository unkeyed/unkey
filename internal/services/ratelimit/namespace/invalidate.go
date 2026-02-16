package namespace

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

// Invalidate removes the namespace from cache by both name and ID.
func (s *service) Invalidate(ctx context.Context, workspaceID string, ns db.FindRatelimitNamespace) {
	s.cache.Remove(ctx,
		cache.ScopedKey{WorkspaceID: workspaceID, Key: ns.ID},
		cache.ScopedKey{WorkspaceID: workspaceID, Key: ns.Name},
	)
}
