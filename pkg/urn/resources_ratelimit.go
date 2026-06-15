package urn

import "fmt"

// ratelimitNamespace builds rate limit namespace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
//
// The namespace is intentionally below the literal "ratelimits" segment so all
// rate limit resources can share one top-level workspace branch.
type ratelimitNamespace struct {
	workspaceID string
	path        string
}

// Override returns a rate limit override resource path.
//
// Subresource:
//
//	ratelimits/namespaces/{namespace_id}
//	└── overrides/{override_id}
func (r ratelimitNamespace) Override(overrideID string) V1 {
	return V1{
		WorkspaceID: r.workspaceID,
		Resource:    fmt.Sprintf("%s/overrides/%s", r.path, overrideID),
	}
}

// Any returns a descendant pattern below this rate limit namespace.
func (r ratelimitNamespace) Any() V1 {
	return V1{
		WorkspaceID: r.workspaceID,
		Resource:    r.path + "/**",
	}
}
