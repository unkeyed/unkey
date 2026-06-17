package urn

import "fmt"

// RatelimitNamespace builds rate limit namespace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
//
// The namespace is intentionally below the literal "ratelimits" segment so all
// rate limit resources can share one top-level workspace branch.
type RatelimitNamespace struct {
	workspaceID string
	path        string
}

// String returns this rate limit namespace resource path.
//
// Subresource:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
func (r RatelimitNamespace) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}.String()
}

// RatelimitOverride is a rate limit override resource path.
type RatelimitOverride struct {
	workspaceID string
	path        string
}

// String returns this rate limit override resource path.
func (r RatelimitOverride) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}.String()
}

// V1 returns this rate limit override as a parsed v1 resource name.
func (r RatelimitOverride) V1() V1 {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}
}

// Override returns a rate limit override resource path.
//
// Subresource:
//
//	ratelimits/namespaces/{namespace_id}
//	└── overrides/{override_id}
func (r RatelimitNamespace) Override(overrideID string) RatelimitOverride {
	return RatelimitOverride{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/overrides/%s", r.path, overrideID)}
}

// Any returns a descendant pattern below this rate limit namespace.
func (r RatelimitNamespace) Any() V1 {
	return V1{
		WorkspaceID: r.workspaceID,
		Resource:    r.path + "/**",
	}
}
