package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateProject authorizes creating a project resource.
type CreateProject struct{}

func (CreateProject) ActionFor(urn.Project) {}
func (CreateProject) String() string        { return "create_project" }

// ReadProject authorizes reading a project resource.
type ReadProject struct{}

func (ReadProject) ActionFor(urn.Project) {}
func (ReadProject) String() string        { return "read_project" }

// UpdateProject authorizes updating a project resource.
type UpdateProject struct{}

func (UpdateProject) ActionFor(urn.Project) {}
func (UpdateProject) String() string        { return "update_project" }

// DeleteProject authorizes deleting a project resource.
type DeleteProject struct{}

func (DeleteProject) ActionFor(urn.Project) {}
func (DeleteProject) String() string        { return "delete_project" }

// CreateApp authorizes creating apps in a project.
//
// Valid resource: urn.App.
type CreateApp struct{}

func (CreateApp) ActionFor(urn.App) {}
func (CreateApp) String() string    { return "create_app" }
