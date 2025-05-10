// Package permissions provides functionality for checking permissions associated with API keys.
// It implements a role-based access control (RBAC) system to determine if a key has
// the required permissions to perform specific actions.
package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/rbac"
)

// PermissionService defines the interface for permission checking operations.
// It provides methods to check if a key has the required permissions.
type PermissionService interface {
	// Check evaluates if a key has the required permissions based on the provided query.
	// It retrieves the permissions associated with the key and evaluates them against
	// the permission query.
	//
	// Returns an evaluation result indicating if the key has the required permissions,
	// and an error if the check fails for any reason.
	Check(ctx context.Context, keyId string, query rbac.PermissionQuery) (rbac.EvaluationResult, error)
}
