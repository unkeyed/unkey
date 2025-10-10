package rbac

import (
	"errors"
	"fmt"
	"strings"
)

// ResourceType represents categories of resources that can be protected
// by the RBAC system. It helps organize permissions by functional area.
type ResourceType string

// ActionType represents operations that can be performed on resources.
// Each resource type can have its own set of applicable actions.
type ActionType string

// Predefined resource types. These constants provide a standardized set of
// resource categories used throughout the system.
const (
	// Api represents API-related resources, such as endpoints, keys, etc.
	Api ResourceType = "api"

	// Ratelimit represents rate limiting resources and configuration
	Ratelimit ResourceType = "ratelimit"

	// Rbac represents permissions and roles management resources
	Rbac ResourceType = "rbac"

	// Identity represents user and identity management resources
	Identity ResourceType = "identity"
)

// Predefined API actions. These constants define operations that can be
// performed on API resources.
const (
	// ReadAPI permits viewing API details
	ReadAPI ActionType = "read_api"

	// CreateAPI permits creating new APIs
	CreateAPI ActionType = "create_api"

	// DeleteAPI permits removing existing APIs
	DeleteAPI ActionType = "delete_api"

	// UpdateAPI permits modifying API configurations
	UpdateAPI ActionType = "update_api"

	// CreateKey permits generating new API keys
	CreateKey ActionType = "create_key"

	// UpdateKey permits modifying existing API keys
	UpdateKey ActionType = "update_key"

	// DeleteKey permits removing API keys
	DeleteKey ActionType = "delete_key"

	// EncryptKey permits encrypting API keys
	EncryptKey ActionType = "encrypt_key"

	// DecryptKey permits decrypting API keys
	DecryptKey ActionType = "decrypt_key"

	// ReadKey permits viewing API key details
	ReadKey ActionType = "read_key"

	// VerifyKey permits verifying API keys
	VerifyKey ActionType = "verify_key"

	// ReadAnalytics permits viewing API analytics
	ReadAnalytics ActionType = "read_analytics"
)

// Predefined rate limiting actions. These constants define operations
// that can be performed on rate limiting resources.
const (
	// Limit permits applying rate limits
	Limit ActionType = "limit"

	// CreateNamespace permits creating rate limit namespaces
	CreateNamespace ActionType = "create_namespace"

	// ReadNamespace permits viewing rate limit namespace details
	ReadNamespace ActionType = "read_namespace"

	// UpdateNamespace permits modifying rate limit namespaces
	UpdateNamespace ActionType = "update_namespace"

	// DeleteNamespace permits removing rate limit namespaces
	DeleteNamespace ActionType = "delete_namespace"

	// SetOverride permits setting rate limit overrides
	SetOverride ActionType = "set_override"

	// ReadOverride permits viewing rate limit override details
	ReadOverride ActionType = "read_override"

	// DeleteOverride permits removing rate limit overrides
	DeleteOverride ActionType = "delete_override"

	// ListOverrides permits viewing rate limit override lists
	ListOverrides ActionType = "list_overrides"
)

// Predefined RBAC actions. These constants define operations that can be
// performed on permission and role resources.
const (
	// CreatePermission permits defining new permissions
	CreatePermission ActionType = "create_permission"

	// UpdatePermission permits modifying existing permissions
	UpdatePermission ActionType = "update_permission"

	// DeletePermission permits removing permissions
	DeletePermission ActionType = "delete_permission"

	// ReadPermission permits viewing permission details
	ReadPermission ActionType = "read_permission"

	// CreateRole permits defining new roles
	CreateRole ActionType = "create_role"

	// UpdateRole permits modifying existing roles
	UpdateRole ActionType = "update_role"

	// DeleteRole permits removing roles
	DeleteRole ActionType = "delete_role"

	// ReadRole permits viewing role details
	ReadRole ActionType = "read_role"

	// AddPermissionToKey permits assigning permissions directly to keys
	AddPermissionToKey ActionType = "add_permission_to_key"

	// RemovePermissionFromKey permits unassigning permissions from keys
	RemovePermissionFromKey ActionType = "remove_permission_from_key"

	// AddRoleToKey permits assigning roles to keys
	AddRoleToKey ActionType = "add_role_to_key"

	// RemoveRoleFromKey permits unassigning roles from keys
	RemoveRoleFromKey ActionType = "remove_role_from_key"

	// AddPermissionToRole permits assigning permissions to roles
	AddPermissionToRole ActionType = "add_permission_to_role"

	// RemovePermissionFromRole permits unassigning permissions from roles
	RemovePermissionFromRole ActionType = "remove_permission_from_role"
)

// Predefined identity actions. These constants define operations that can be
// performed on identity resources.
const (
	// CreateIdentity permits creating new identities
	CreateIdentity ActionType = "create_identity"

	// ReadIdentity permits viewing identity details
	ReadIdentity ActionType = "read_identity"

	// UpdateIdentity permits modifying existing identities
	UpdateIdentity ActionType = "update_identity"

	// DeleteIdentity permits removing identities
	DeleteIdentity ActionType = "delete_identity"
)

// Tuple represents a specific permission as a combination of resource type,
// resource ID, and action. It forms the basic unit of permission definition
// in the RBAC system.
//
// Example:
//
//	// Permission to read a specific API
//	readApiPermission := rbac.Tuple{
//	    ResourceType: rbac.Api,
//	    ResourceID:   "api1",
//	    Action:       rbac.ReadAPI,
//	}
type Tuple struct {
	// ResourceType defines the category of resource being accessed
	ResourceType ResourceType

	// ResourceID identifies the specific resource instance
	ResourceID string

	// Action specifies the operation being performed on the resource
	Action ActionType
}

// String converts a Tuple to its string representation in the format
// "resourceType.resourceID.action" (e.g., "api.api1.read_api").
//
// This string format is used when storing and comparing permissions.
func (t Tuple) String() string {
	return fmt.Sprintf("%s.%s.%s", t.ResourceType, t.ResourceID, t.Action)
}

// TupleFromString parses a string in the format "resourceType.resourceID.action"
// into a Tuple. Returns an error if the string format is invalid.
//
// Example:
//
//	tuple, err := rbac.TupleFromString("api.api1.read_api")
//	if err != nil {
//	    log.Fatalf("Invalid permission format: %v", err)
//	}
func TupleFromString(s string) (Tuple, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Tuple{}, errors.New("invalid tuple format")

	}
	tuple := Tuple{
		ResourceType: ResourceType(parts[0]),
		ResourceID:   parts[1],
		Action:       ActionType(parts[2]),
	}
	return tuple, nil
}
