package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadApp authorizes reading an app resource.
type ReadApp struct{}

func (ReadApp) ActionFor(urn.App) {}
func (ReadApp) String() string    { return "read_app" }

// UpdateApp authorizes updating an app resource.
type UpdateApp struct{}

func (UpdateApp) ActionFor(urn.App) {}
func (UpdateApp) String() string    { return "update_app" }

// DeleteApp authorizes deleting an app resource.
type DeleteApp struct{}

func (DeleteApp) ActionFor(urn.App) {}
func (DeleteApp) String() string    { return "delete_app" }

// CreateEnvironment authorizes creating environments in an app.
//
// Valid resource: urn.App.
type CreateEnvironment struct{}

func (CreateEnvironment) ActionFor(urn.App) {}
func (CreateEnvironment) String() string    { return "create_environment" }
