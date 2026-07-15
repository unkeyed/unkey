package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateApp authorizes creating apps in a project.
//
// Valid resource: urn.Project.
type CreateApp struct{}

func (CreateApp) ActionFor(urn.Project) {}
func (CreateApp) String() string        { return "create_app" }
