package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateIdentity authorizes creating identity resources.
//
// Valid resource: urn.Identity.
type CreateIdentity struct{}

func (CreateIdentity) ActionFor(urn.Identity) {}
func (CreateIdentity) String() string         { return "create_identity" }

// ReadIdentity authorizes reading identity resources.
//
// Valid resource: urn.Identity.
type ReadIdentity struct{}

func (ReadIdentity) ActionFor(urn.Identity) {}
func (ReadIdentity) String() string         { return "read_identity" }

// WriteIdentity authorizes creating or updating an identity.
type WriteIdentity struct{}

func (WriteIdentity) ActionFor(urn.Identity) {}
func (WriteIdentity) String() string         { return "write_identity" }

// UpdateIdentity authorizes updating identity resources.
//
// Valid resource: urn.Identity.
type UpdateIdentity struct{}

func (UpdateIdentity) ActionFor(urn.Identity) {}
func (UpdateIdentity) String() string         { return "update_identity" }

// DeleteIdentity authorizes deleting identity resources.
//
// Valid resource: urn.Identity.
type DeleteIdentity struct{}

func (DeleteIdentity) ActionFor(urn.Identity) {}
func (DeleteIdentity) String() string         { return "delete_identity" }
