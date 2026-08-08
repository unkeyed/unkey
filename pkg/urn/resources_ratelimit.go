package urn

import "fmt"

// RatelimitNamespace builds rate limit namespace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── ratelimits/namespaces/{namespace_id}
//
// The namespace is intentionally below the literal "ratelimits" segment so all
// rate limit resources can share one branch beneath their parent.
type RatelimitNamespace struct {
	workspaceID string
	projectID   string
	namespaceID string
}

// String returns this rate limit namespace resource path.
func (r RatelimitNamespace) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("projects/%s/ratelimits/namespaces/%s", r.projectID, r.namespaceID)}.String()
}

// RatelimitOverride is a rate limit override resource path.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── ratelimits/namespaces/{namespace_id}
//	        └── overrides/{override_id}
type RatelimitOverride struct {
	workspaceID string
	projectID   string
	namespaceID string
	overrideID  string
}

// String returns this rate limit override resource path.
func (r RatelimitOverride) String() string {
	return r.V1().String()
}

// V1 returns this rate limit override as a parsed v1 resource name.
func (r RatelimitOverride) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("projects/%s/ratelimits/namespaces/%s/overrides/%s", r.projectID, r.namespaceID, r.overrideID)}
}

// Override returns a rate limit override resource path.
func (r RatelimitNamespace) Override(overrideID string) RatelimitOverride {
	return RatelimitOverride{workspaceID: r.workspaceID, projectID: r.projectID, namespaceID: r.namespaceID, overrideID: overrideID}
}

// Any returns a descendant pattern below this rate limit namespace.
func (r RatelimitNamespace) Any() V1 {
	return V1{
		WorkspaceID: r.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/ratelimits/namespaces/%s/**", r.projectID, r.namespaceID),
	}
}
