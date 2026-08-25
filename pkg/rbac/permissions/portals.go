package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreatePortal authorizes creating portals.
//
// Valid resource: urn.Portal. Grants use a wildcard portal id because the
// portal does not exist when the request is authorized.
type CreatePortal struct{}

func (CreatePortal) ActionFor(urn.Portal) {}
func (CreatePortal) String() string       { return "create_portal" }

// ReadPortal authorizes reading a specific portal.
//
// Valid resource: urn.Portal.
type ReadPortal struct{}

func (ReadPortal) ActionFor(urn.Portal) {}
func (ReadPortal) String() string       { return "read_portal" }

// UpdatePortal authorizes updating a specific portal.
//
// Valid resource: urn.Portal.
type UpdatePortal struct{}

func (UpdatePortal) ActionFor(urn.Portal) {}
func (UpdatePortal) String() string       { return "update_portal" }

// DeletePortal authorizes deleting a specific portal.
//
// Valid resource: urn.Portal.
type DeletePortal struct{}

func (DeletePortal) ActionFor(urn.Portal) {}
func (DeletePortal) String() string       { return "delete_portal" }

// CreatePortalSession authorizes minting a session for an end user of a
// specific portal. It is separate from the portal management actions so a key
// can mint sessions without being able to change portals.
//
// Valid resource: urn.Portal.
type CreatePortalSession struct{}

func (CreatePortalSession) ActionFor(urn.Portal) {}
func (CreatePortalSession) String() string       { return "create_portal_session" }
