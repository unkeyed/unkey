// Package permissions validates actions for URN resources.
package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// Action identifies an operation on a resource name.
type Action = urn.PermissionAction

const (
	// Read authorizes reading a resource, except root keys.
	Read = urn.PermissionRead
	// Write authorizes creating or updating a resource.
	Write = urn.PermissionWrite
	// Delete authorizes deleting a resource.
	Delete = urn.PermissionDelete
	// Decrypt authorizes decrypting key data.
	Decrypt = urn.PermissionDecrypt
	// Verify authorizes verifying a key.
	Verify = urn.PermissionVerify
	// Limit authorizes using a rate limit namespace.
	Limit = urn.PermissionLimit
)

// Wildcard is the action used by the global administrator permission.
const Wildcard = "*"

// IsValid reports whether action is supported by resource. It validates the
// complete resource name, so a zero value or manually constructed invalid
// [urn.V1] returns false.
func IsValid(resource urn.V1, action Action) bool {
	return resource.SupportsPermissionAction(action)
}
