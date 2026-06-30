package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// AddPermissionToKey authorizes attaching permissions to key resources.
//
// Valid resource: urn.Key.
type AddPermissionToKey struct{}

func (AddPermissionToKey) ActionFor(urn.Key) {}
func (AddPermissionToKey) String() string    { return "add_permission_to_key" }

// RemovePermissionFromKey authorizes detaching permissions from key resources.
//
// Valid resource: urn.Key.
type RemovePermissionFromKey struct{}

func (RemovePermissionFromKey) ActionFor(urn.Key) {}
func (RemovePermissionFromKey) String() string    { return "remove_permission_from_key" }

// AddRoleToKey authorizes attaching roles to key resources.
//
// Valid resource: urn.Key.
type AddRoleToKey struct{}

func (AddRoleToKey) ActionFor(urn.Key) {}
func (AddRoleToKey) String() string    { return "add_role_to_key" }

// RemoveRoleFromKey authorizes detaching roles from key resources.
//
// Valid resource: urn.Key.
type RemoveRoleFromKey struct{}

func (RemoveRoleFromKey) ActionFor(urn.Key) {}
func (RemoveRoleFromKey) String() string    { return "remove_role_from_key" }
