package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadProject authorizes reading a project.
type ReadProject struct{}

func (ReadProject) ActionFor(urn.Project) {}
func (ReadProject) String() string        { return "read_project" }

// WriteProject authorizes creating or updating a project.
type WriteProject struct{}

func (WriteProject) ActionFor(urn.Project) {}
func (WriteProject) String() string        { return "write_project" }

// DeleteProject authorizes deleting a specific project.
//
// Valid resource: urn.Project.
type DeleteProject struct{}

func (DeleteProject) ActionFor(urn.Project) {}
func (DeleteProject) String() string        { return "delete_project" }
