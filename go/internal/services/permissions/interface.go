package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/rbac"
)

type PermissionService interface {
	// If the user does not have the required permissions, an error is returned.
	// The returned error will have a code of codes.Auth.Authorization.InsufficientPermissions.URN()
	// and can be returned in a zen route as is.
	Check(ctx context.Context, keyId string, query rbac.PermissionQuery) error
}
