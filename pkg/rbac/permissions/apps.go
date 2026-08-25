package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateApp authorizes creating apps.
//
// Valid resource: urn.App. Grants use a wildcard app id because the app does
// not exist when the request is authorized.
type CreateApp struct{}

func (CreateApp) ActionFor(urn.App) {}
func (CreateApp) String() string    { return "create_app" }

// ReadApp authorizes reading an app resource.
type ReadApp struct{}

func (ReadApp) ActionFor(urn.App) {}
func (ReadApp) String() string    { return "read_app" }

// UpdateApp authorizes updating an app resource.
type UpdateApp struct{}

func (UpdateApp) ActionFor(urn.App) {}
func (UpdateApp) String() string    { return "update_app" }

// DeleteApp authorizes deleting a specific app.
//
// Valid resource: urn.App.
type DeleteApp struct{}

func (DeleteApp) ActionFor(urn.App) {}
func (DeleteApp) String() string    { return "delete_app" }
