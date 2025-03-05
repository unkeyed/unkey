package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

func (s *service) Check(ctx context.Context, keyID string, query rbac.PermissionQuery) (rbac.EvaluationResult, error) {

	permissions, err := db.Query.FindPermissionsForKey(ctx, s.db.RO(), db.FindPermissionsForKeyParams{
		KeyID: keyID,
	})

	if err != nil {
		return rbac.EvaluationResult{}, fault.Wrap(err, fault.WithDesc("unable to laod permissions from db", ""))
	}

	return s.rbac.EvaluatePermissions(query, permissions)
}
