package urn

import "fmt"

// Environment builds environment resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
type Environment struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

// String returns this environment resource path.
func (e Environment) String() string {
	return V1{WorkspaceID: e.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s", e.projectID, e.appID, e.environmentID)}.String()
}

// Deployment returns builders for deployment resource paths.
func (e Environment) Deployment(deploymentID string) Deployment {
	return Deployment{workspaceID: e.workspaceID, projectID: e.projectID, appID: e.appID, environmentID: e.environmentID, deploymentID: deploymentID}
}

// Domain returns a domain resource path.
func (e Environment) Domain(domainID string) Domain {
	return Domain{workspaceID: e.workspaceID, projectID: e.projectID, appID: e.appID, environmentID: e.environmentID, domainID: domainID}
}

// Variable is an environment variable resource path.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── variables/{variable_id}
type Variable struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	variableID    string
}

// String returns this environment variable resource path.
func (v Variable) String() string {
	return V1{WorkspaceID: v.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/variables/%s", v.projectID, v.appID, v.environmentID, v.variableID)}.String()
}

// Variable returns a variable resource path.
func (e Environment) Variable(variableID string) Variable {
	return Variable{workspaceID: e.workspaceID, projectID: e.projectID, appID: e.appID, environmentID: e.environmentID, variableID: variableID}
}

// Gateway returns the environment's gateway resource namespace.
func (e Environment) Gateway() Gateway {
	return Gateway{workspaceID: e.workspaceID, projectID: e.projectID, appID: e.appID, environmentID: e.environmentID}
}

// Any returns a descendant pattern below this environment.
func (e Environment) Any() V1 {
	return V1{
		WorkspaceID: e.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/apps/%s/environments/%s/**", e.projectID, e.appID, e.environmentID),
	}
}
