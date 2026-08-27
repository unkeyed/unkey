package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadIdentity authorizes reading an identity.
type ReadIdentity struct{}

func (ReadIdentity) ActionFor(urn.Identity) {}
func (ReadIdentity) String() string         { return "read_identity" }

// WriteIdentity authorizes creating or updating an identity.
type WriteIdentity struct{}

func (WriteIdentity) ActionFor(urn.Identity) {}
func (WriteIdentity) String() string         { return "write_identity" }

// DeleteIdentity authorizes deleting an identity.
type DeleteIdentity struct{}

func (DeleteIdentity) ActionFor(urn.Identity) {}
func (DeleteIdentity) String() string         { return "delete_identity" }
