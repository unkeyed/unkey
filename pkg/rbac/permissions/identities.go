package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateIdentity authorizes creating identity resources.
//
// Valid resource: urn.ProjectIdentity.
type CreateIdentity struct{}

func (CreateIdentity) ActionFor(urn.ProjectIdentity) {}
func (CreateIdentity) String() string                { return "create_identity" }

// ReadIdentity authorizes reading identity resources.
//
// Valid resource: urn.ProjectIdentity.
type ReadIdentity struct{}

func (ReadIdentity) ActionFor(urn.ProjectIdentity) {}
func (ReadIdentity) String() string                { return "read_identity" }

// UpdateIdentity authorizes updating identity resources.
//
// Valid resource: urn.ProjectIdentity.
type UpdateIdentity struct{}

func (UpdateIdentity) ActionFor(urn.ProjectIdentity) {}
func (UpdateIdentity) String() string                { return "update_identity" }

// DeleteIdentity authorizes deleting identity resources.
//
// Valid resource: urn.ProjectIdentity.
type DeleteIdentity struct{}

func (DeleteIdentity) ActionFor(urn.ProjectIdentity) {}
func (DeleteIdentity) String() string                { return "delete_identity" }
