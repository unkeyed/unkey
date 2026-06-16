package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateEnvironment authorizes creating environments in an app.
//
// Valid resource: urn.App.
type CreateEnvironment struct{}

func (CreateEnvironment) ActionFor(urn.App) {}
func (CreateEnvironment) String() string    { return "create_environment" }
