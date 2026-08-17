package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateApp authorizes creating apps.
//
// Valid resource: urn.App. Grants use a wildcard app id because the app does
// not exist when the request is authorized.
type CreateApp struct{}

func (CreateApp) ActionFor(urn.App) {}
func (CreateApp) String() string    { return "create_app" }

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
