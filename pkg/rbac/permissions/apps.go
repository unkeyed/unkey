package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadApp authorizes reading an app.
type ReadApp struct{}

func (ReadApp) ActionFor(urn.App) {}
func (ReadApp) String() string    { return "read_app" }

// WriteApp authorizes creating or updating an app.
type WriteApp struct{}

func (WriteApp) ActionFor(urn.App) {}
func (WriteApp) String() string    { return "write_app" }

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
