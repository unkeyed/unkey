package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateIdentity authorizes creating identity resources.
//
// Valid resource: urn.Project.
type CreateIdentity struct{}

func (CreateIdentity) ActionFor(urn.Project) {}
func (CreateIdentity) String() string        { return "create_identity" }

// ReadIdentity authorizes reading identity resources.
//
// Valid resource: urn.Project.
type ReadIdentity struct{}

func (ReadIdentity) ActionFor(urn.Project) {}
func (ReadIdentity) String() string        { return "read_identity" }

// UpdateIdentity authorizes updating identity resources.
//
// Valid resource: urn.Project.
type UpdateIdentity struct{}

func (UpdateIdentity) ActionFor(urn.Project) {}
func (UpdateIdentity) String() string        { return "update_identity" }

// DeleteIdentity authorizes deleting identity resources.
//
// Valid resource: urn.Project.
type DeleteIdentity struct{}

func (DeleteIdentity) ActionFor(urn.Project) {}
func (DeleteIdentity) String() string        { return "delete_identity" }
