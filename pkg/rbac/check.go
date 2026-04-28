package rbac

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// HasAnyPermission reports whether granted contains at least one permission
// for the given resource type and action, regardless of the resource ID.
// It looks for entries shaped like "<resourceType>.<anything>.<action>".
//
// Used by the verify_key fast path to short-circuit when the caller has no
// matching permissions at all, avoiding an unnecessary key lookup.
func HasAnyPermission(granted []string, resourceType ResourceType, action ActionType) bool {
	prefix := string(resourceType) + "."
	suffix := "." + string(action)
	for _, perm := range granted {
		if strings.HasPrefix(perm, prefix) && strings.HasSuffix(perm, suffix) {
			return true
		}
	}
	return false
}

// Check evaluates query against the granted permission strings and returns
// a fault.Error tagged with codes.Auth.Authorization.InsufficientPermissions
// when the query is not satisfied. Returns nil on success.
//
// Handlers should use this in place of the verifier-bound permission check
// so the same call works for root-key auth and JWT auth.
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
