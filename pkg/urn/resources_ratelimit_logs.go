package urn

import "fmt"

const ratelimitLogsPathFormat = "projects/%s/ratelimits/namespaces/%s/logs"

var ratelimitLogsPattern = compileResourcePattern(ratelimitLogsPathFormat)

// RatelimitLogs builds rate limit log resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── ratelimits/namespaces/{namespace_id}
//	        └── logs
type RatelimitLogs struct {
	WorkspaceID string
	ProjectID   string
	NamespaceID string
}

// String returns the complete URN for these rate limit logs.
func (r RatelimitLogs) String() string {
	return V1{
		WorkspaceID: r.WorkspaceID,
		Resource:    fmt.Sprintf(ratelimitLogsPathFormat, r.ProjectID, r.NamespaceID),
	}.String()
}

// ParseRatelimitLogs parses:
//
//	unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/logs
//
// into:
//
//	RatelimitLogs{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		NamespaceID: "ns_123",
//	}
//
// Resource ID positions may contain "*".
func ParseRatelimitLogs(urn string) (RatelimitLogs, error) {
	matches := ratelimitLogsPattern.FindStringSubmatch(urn)
	if matches == nil {
		return RatelimitLogs{}, fmt.Errorf("%w: resource does not match rate limit logs", ErrInvalidResourceName)
	}

	return RatelimitLogs{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		NamespaceID: matches[3],
	}, nil
}
