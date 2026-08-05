package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateApp authorizes creating an app resource.
type CreateApp struct{}

func (CreateApp) ActionFor(urn.App) {}
func (CreateApp) String() string    { return "create_app" }

// ReadApp authorizes reading an app resource.
type ReadApp struct{}

func (ReadApp) ActionFor(urn.App) {}
func (ReadApp) String() string    { return "read_app" }

// SetRepository authorizes setting or clearing an app repository.
type SetRepository struct{}

func (SetRepository) ActionFor(urn.App) {}
func (SetRepository) String() string    { return "set_repository" }

// UpdateApp authorizes updating an app resource.
type UpdateApp struct{}

func (UpdateApp) ActionFor(urn.App) {}
func (UpdateApp) String() string    { return "update_app" }

// DeleteApp authorizes deleting an app resource.
type DeleteApp struct{}

func (DeleteApp) ActionFor(urn.App) {}
func (DeleteApp) String() string    { return "delete_app" }
