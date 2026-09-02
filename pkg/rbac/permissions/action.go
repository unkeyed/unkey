// Package permissions defines actions for canonical URN resources.
package permissions

// Action identifies an operation on a canonical resource name. The resource
// path identifies the resource type.
type Action string

// String returns the serialized permission action.
func (a Action) String() string {
	return string(a)
}

const (
	// Read authorizes reading a resource. It applies to every concrete resource
	// in the canonical permission catalog.
	Read Action = "read"

	// Write authorizes creating or updating a resource. It applies to every
	// concrete resource except deployment logs, gateway logs, keyspace logs,
	// and rate limit logs.
	Write Action = "write"

	// Delete authorizes deleting a resource. It applies to every concrete
	// resource except deployment logs, gateway logs, keyspace logs, and rate
	// limit logs.
	Delete Action = "delete"

	// Decrypt authorizes decrypting protected resource data. It applies only to
	// keys.
	Decrypt Action = "decrypt"

	// Verify authorizes verifying a resource. It applies only to keys.
	Verify Action = "verify"

	// Limit authorizes using a rate limit namespace. It applies only to rate
	// limit namespaces.
	Limit Action = "limit"
)

// Wildcard is the action used by the global admin permission.
const Wildcard = "*"
