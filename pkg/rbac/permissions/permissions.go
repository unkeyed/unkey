package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadPermission authorizes reading a permission definition.
type ReadPermission struct{}

func (ReadPermission) ActionFor(urn.Permission) {}
func (ReadPermission) String() string           { return "read_permission" }

// WritePermission authorizes creating or updating a permission definition.
type WritePermission struct{}

func (WritePermission) ActionFor(urn.Permission) {}
func (WritePermission) String() string           { return "write_permission" }

// DeletePermission authorizes deleting a permission definition.
type DeletePermission struct{}

func (DeletePermission) ActionFor(urn.Permission) {}
func (DeletePermission) String() string           { return "delete_permission" }
