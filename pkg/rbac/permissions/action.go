// Package permissions defines actions for URN resources.
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
