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

// DeleteApp authorizes deleting a specific app.
//
// Valid resource: urn.App.
type DeleteApp struct{}

func (DeleteApp) ActionFor(urn.App) {}
func (DeleteApp) String() string    { return "delete_app" }
