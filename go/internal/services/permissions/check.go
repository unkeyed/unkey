package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

func (s *service) Check(ctx context.Context, keyID string, query rbac.PermissionQuery) (rbac.EvaluationResult, error) {

	permissions, err := s.cache.SWR(ctx, keyID, func(ctx context.Context) ([]string, error) {
		return db.Query.FindPermissionsForKey(ctx, s.db.RO(), db.FindPermissionsForKeyParams{
			KeyID: keyID,
		})
	}, func(err error) cache.CacheHit {
		if err == nil {
			return cache.Hit
		}
		return cache.Miss

	})

	if err != nil {
		return rbac.EvaluationResult{}, fault.Wrap(err, fault.WithDesc("unable to laod permissions from db", ""))
	}

	return s.rbac.EvaluatePermissions(query, permissions)
}
