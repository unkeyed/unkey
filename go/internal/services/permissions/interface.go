package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/rbac"
)

type PermissionService interface {
	Check(ctx context.Context, keyId string, query rbac.PermissionQuery) (rbac.EvaluationResult, error)
}
