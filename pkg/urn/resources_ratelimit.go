package urn

import "fmt"

const ratelimitNamespacePathFormat = "projects/%s/ratelimits/namespaces/%s"

var ratelimitNamespacePattern = compileResourcePattern(ratelimitNamespacePathFormat)

// RatelimitNamespace builds rate limit namespace resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── ratelimits/namespaces/{namespace_id}
//	        ├── logs
//	        └── overrides/{override_id}
type RatelimitNamespace struct {
	WorkspaceID string
	ProjectID   string
	NamespaceID string
}

// String returns the complete URN for this rate limit namespace.
//
// Subresource:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
func (r RatelimitNamespace) String() string {
	return V1{
		WorkspaceID: r.WorkspaceID,
		Resource:    fmt.Sprintf(ratelimitNamespacePathFormat, r.ProjectID, r.NamespaceID),
	}.String()
}

// ParseRatelimitNamespace parses:
//
//	unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123
//
// into:
//
//	RatelimitNamespace{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		NamespaceID: "ns_123",
//	}
//
// Resource ID positions may contain "*".
func ParseRatelimitNamespace(urn string) (RatelimitNamespace, error) {
	matches := ratelimitNamespacePattern.FindStringSubmatch(urn)
	if matches == nil {
		return RatelimitNamespace{}, fmt.Errorf("%w: resource does not match rate limit namespace", ErrInvalidResourceName)
	}
	return RatelimitNamespace{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		NamespaceID: matches[3],
	}, nil
}

// Logs returns the rate limit log resource path.
//
// Subresource:
//
//	ratelimits/namespaces/{namespace_id}
//	└── logs
func (r RatelimitNamespace) Logs() RatelimitLogs {
	return RatelimitLogs{
		WorkspaceID: r.WorkspaceID,
		ProjectID:   r.ProjectID,
		NamespaceID: r.NamespaceID,
	}
}

// Override returns a rate limit override resource path.
//
// Subresource:
//
//	ratelimits/namespaces/{namespace_id}
//	└── overrides/{override_id}
func (r RatelimitNamespace) Override(overrideID string) RatelimitOverride {
	return RatelimitOverride{
		WorkspaceID: r.WorkspaceID,
		ProjectID:   r.ProjectID,
		NamespaceID: r.NamespaceID,
		OverrideID:  overrideID,
	}
}
