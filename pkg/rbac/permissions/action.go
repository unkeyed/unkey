// Package permissions defines actions for canonical URN resources.
package permissions

import "fmt"

// Action is an operation that can be applied to a canonical resource name.
// The resource path identifies the resource type.
type Action interface {
	fmt.Stringer
}

// Read authorizes reading a resource.
type Read struct{}

func (Read) String() string { return "read" }

// Write authorizes creating or updating a resource.
type Write struct{}

func (Write) String() string { return "write" }

// Delete authorizes deleting a resource.
type Delete struct{}

func (Delete) String() string { return "delete" }

// Decrypt authorizes decrypting protected resource data.
type Decrypt struct{}

func (Decrypt) String() string { return "decrypt" }

// Verify authorizes verifying a resource.
type Verify struct{}

func (Verify) String() string { return "verify" }

// Limit authorizes using a rate limit namespace.
type Limit struct{}

func (Limit) String() string { return "limit" }

// Wildcard is the action used by the global admin permission.
const Wildcard = "*"
