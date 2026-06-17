package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateDeployment authorizes creating deployments in an environment.
//
// Valid resource: urn.Environment.
type CreateDeployment struct{}

func (CreateDeployment) ActionFor(urn.Environment) {}
func (CreateDeployment) String() string            { return "create_deployment" }

// CreateDomain authorizes creating domains in an environment.
//
// Valid resource: urn.Environment.
type CreateDomain struct{}

func (CreateDomain) ActionFor(urn.Environment) {}
func (CreateDomain) String() string            { return "create_domain" }

// CreateVariable authorizes creating variables in an environment.
//
// Valid resource: urn.Environment.
type CreateVariable struct{}

func (CreateVariable) ActionFor(urn.Environment) {}
func (CreateVariable) String() string            { return "create_variable" }
