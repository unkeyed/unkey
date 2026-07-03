package rbac

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Check evaluates granted permissions and returns an authorization fault on denial.
func Check(query PermissionQuery, granted []string) error {
	result, err := New().EvaluatePermissions(query, granted)
	if err != nil {
		return err
	}
	if result.Valid {
		return nil
	}

	message := result.Message
	if message == "" {
		message = "Insufficient permissions to access this resource."
	}

	return fault.New("insufficient permissions",
		fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
		fault.Internal(message),
		fault.Public(message),
	)
}
