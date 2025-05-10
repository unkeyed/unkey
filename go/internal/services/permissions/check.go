package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

// Check evaluates if a key has the required permissions based on the provided query.
// It retrieves the permissions associated with the key from the cache or database,
// then evaluates them against the permission query using the RBAC system.
//
// The method first attempts to retrieve permissions from the cache. If not found,
// it queries the database and updates the cache with the result.
//
// Parameters:
//   - ctx: Context for the operation, allowing for cancellation and timeout
//   - keyID: The ID of the key whose permissions are being checked
//   - query: The permission query to evaluate against the key's permissions
//
// Returns:
//   - rbac.EvaluationResult: The result of the permission evaluation
//   - error: Any error that occurred during the permission check
//
// This method is used throughout the system whenever permission-based authorization
// is needed, such as when handling API requests that require specific permissions.
func (s *service) Check(ctx context.Context, keyID string, query rbac.PermissionQuery) (rbac.EvaluationResult, error) {

	ctx, span := tracing.Start(ctx, "permissions.Check")
	defer span.End()

	permissions, err := s.cache.SWR(ctx, keyID, func(ctx context.Context) ([]string, error) {
		return db.Query.FindPermissionsForKey(ctx, s.db.RO(), db.FindPermissionsForKeyParams{
			KeyID: keyID,
		})

	}, func(err error) cache.Op {
		if err == nil {
			return cache.WriteValue
		}
		return cache.Noop

	})

	if err != nil {
		return rbac.EvaluationResult{}, fault.Wrap(err, fault.WithDesc("unable to laod permissions from db", ""))
	}

	return s.rbac.EvaluatePermissions(query, permissions)
}
