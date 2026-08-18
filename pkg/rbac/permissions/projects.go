package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateProject authorizes creating projects in a workspace.
//
// Valid resource: urn.Project.
type CreateProject struct{}

func (CreateProject) ActionFor(urn.Project) {}
func (CreateProject) String() string        { return "create_project" }

// UpdateProject authorizes updating a specific project.
//
// Valid resource: urn.Project.
type UpdateProject struct{}

func (UpdateProject) ActionFor(urn.Project) {}
func (UpdateProject) String() string        { return "update_project" }

// DeleteProject authorizes deleting a specific project.
//
// Valid resource: urn.Project.
type DeleteProject struct{}

func (DeleteProject) ActionFor(urn.Project) {}
func (DeleteProject) String() string        { return "delete_project" }
