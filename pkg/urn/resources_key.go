package urn

import "fmt"

const keyPathFormat = "projects/%s/keyspaces/%s/keys/%s"

var keyPattern = compileResourcePattern(keyPathFormat)

// Key is a key resource path.
type Key struct {
	WorkspaceID string
	ProjectID   string
	KeyspaceID  string
	KeyID       string
}

// String returns the complete URN for this key.
func (k Key) String() string {
	return V1{
		WorkspaceID: k.WorkspaceID,
		Resource:    fmt.Sprintf(keyPathFormat, k.ProjectID, k.KeyspaceID, k.KeyID),
	}.String()
}

// ParseKey parses:
//
//	unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123
//
// into:
//
//	Key{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		KeyspaceID:  "ks_123",
//		KeyID:       "key_123",
//	}
//
// Resource ID positions may contain "*".
func ParseKey(urn string) (Key, error) {
	matches := keyPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Key{}, fmt.Errorf("%w: resource does not match key", ErrInvalidResourceName)
	}
	return Key{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		KeyspaceID:  matches[3],
		KeyID:       matches[4],
	}, nil
}

// V1 returns this key as a parsed v1 resource name.
func (k Key) V1() V1 {
	return V1{
		WorkspaceID: k.WorkspaceID,
		Resource:    fmt.Sprintf(keyPathFormat, k.ProjectID, k.KeyspaceID, k.KeyID),
	}
}
