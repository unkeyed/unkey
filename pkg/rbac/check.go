package rbac

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// HasAnyPermission reports whether any granted permission matches the resource action.
func HasAnyPermission(granted []string, resourceType ResourceType, action ActionType) bool {
	prefix := string(resourceType) + "."
	suffix := "." + string(action)
	for _, permission := range granted {
		if strings.HasPrefix(permission, prefix) && strings.HasSuffix(permission, suffix) {
			return true
		}
		unkeyPermission, err := parseUrnPermission(permission)
		if err != nil {
			continue
		}
		if unkeyPermission.Action == action || unkeyPermission.Action == "*" {
			return true
		}
	}
	return false
}

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
