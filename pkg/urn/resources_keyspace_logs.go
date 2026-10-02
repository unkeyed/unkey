package urn

import "fmt"

const keyspaceLogsPathFormat = "projects/%s/keyspaces/%s/logs"

var keyspaceLogsPattern = compileResourcePattern(keyspaceLogsPathFormat)

// KeyspaceLogs builds keyspace log resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── keyspaces/{keyspace_id}
//	        └── logs
type KeyspaceLogs struct {
	WorkspaceID string
	ProjectID   string
	KeyspaceID  string
}

// String returns the complete URN for these keyspace logs.
func (k KeyspaceLogs) String() string {
	return V1{
		WorkspaceID: k.WorkspaceID,
		Resource:    fmt.Sprintf(keyspaceLogsPathFormat, k.ProjectID, k.KeyspaceID),
	}.String()
}

// ParseKeyspaceLogs parses:
//
//	unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/logs
//
// into:
//
//	KeyspaceLogs{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		KeyspaceID:  "ks_123",
//	}
//
// Resource ID positions may contain "*".
func ParseKeyspaceLogs(urn string) (KeyspaceLogs, error) {
	matches := keyspaceLogsPattern.FindStringSubmatch(urn)
	if matches == nil {
		return KeyspaceLogs{}, fmt.Errorf("%w: resource does not match keyspace logs", ErrInvalidResourceName)
	}
	return KeyspaceLogs{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		KeyspaceID:  matches[3],
	}, nil
}
