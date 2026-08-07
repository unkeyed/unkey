package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// DeleteApp authorizes deleting a specific app.
//
// Valid resource: urn.App.
type DeleteApp struct{}

func (DeleteApp) ActionFor(urn.App) {}
func (DeleteApp) String() string    { return "delete_app" }

// CreateEnvironment authorizes creating environments in an app.
//
// Valid resource: urn.App.
type CreateEnvironment struct{}

func (CreateEnvironment) ActionFor(urn.App) {}
func (CreateEnvironment) String() string    { return "create_environment" }
