package urn

import "fmt"

const environmentPathFormat = "projects/%s/apps/%s/environments/%s"

var environmentPattern = compileResourcePattern(environmentPathFormat)

// Environment builds environment resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            ├── deployments/{deployment_id}
//	            ├── domains/{domain_id}
//	            ├── variables/{variable_id}
//	            └── gateway
type Environment struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
}

// String returns the complete URN for this environment.
//
// Subresource:
//
//	apps/{app_id}
//	└── environments/{environment_id}
func (e Environment) String() string {
	return V1{
		WorkspaceID: e.WorkspaceID,
		Resource:    fmt.Sprintf(environmentPathFormat, e.ProjectID, e.AppID, e.EnvironmentID),
	}.String()
}

// ParseEnvironment parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123
//
// into:
//
//	Environment{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//	}
//
// Resource ID positions may contain "*".
func ParseEnvironment(urn string) (Environment, error) {
	matches := environmentPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Environment{}, fmt.Errorf("%w: resource does not match environment", ErrInvalidResourceName)
	}
	return Environment{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
	}, nil
}

// Deployment returns builders for deployment resource paths.
//
// Subresource:
//
//	environments/{environment_id}
//	└── deployments/{deployment_id}
func (e Environment) Deployment(deploymentID string) Deployment {
	return Deployment{
		WorkspaceID:   e.WorkspaceID,
		ProjectID:     e.ProjectID,
		AppID:         e.AppID,
		EnvironmentID: e.EnvironmentID,
		DeploymentID:  deploymentID,
	}
}

// Domain returns a domain resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── domains/{domain_id}
func (e Environment) Domain(domainID string) Domain {
	return Domain{
		WorkspaceID:   e.WorkspaceID,
		ProjectID:     e.ProjectID,
		AppID:         e.AppID,
		EnvironmentID: e.EnvironmentID,
		DomainID:      domainID,
	}
}

// Gateway returns builders for gateway resource paths.
//
// Subresource:
//
//	environments/{environment_id}
//	└── gateway
func (e Environment) Gateway() Gateway {
	return Gateway{
		WorkspaceID:   e.WorkspaceID,
		ProjectID:     e.ProjectID,
		AppID:         e.AppID,
		EnvironmentID: e.EnvironmentID,
	}
}

// Variable returns an environment variable resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── variables/{variable_id}
func (e Environment) Variable(variableID string) EnvironmentVariable {
	return EnvironmentVariable{
		WorkspaceID:   e.WorkspaceID,
		ProjectID:     e.ProjectID,
		AppID:         e.AppID,
		EnvironmentID: e.EnvironmentID,
		VariableID:    variableID,
	}
}
