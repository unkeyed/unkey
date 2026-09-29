package db

import "database/sql"

// CachedKeyData embeds FindKeyForVerificationRow and adds pre-processed data for caching.
// This struct is stored in the cache to avoid redundant parsing operations.
type CachedKeyData struct {
	FindKeyForVerificationRow
	ParsedIPWhitelist map[string]struct{} // Pre-parsed IP addresses for O(1) lookup
	Roles             []string
	Permissions       []string
	RatelimitConfigs  map[string]KeyFindForVerificationRatelimit
}

// CachedRootKeyData contains the root-key fields used during authentication.
type CachedRootKeyData struct {
	ID                  string
	KeyAuthID           string
	WorkspaceID         string
	ForWorkspaceID      string
	Name                sql.NullString
	Expires             sql.NullTime
	Enabled             bool
	ApiDeletedAtM       sql.NullInt64
	WorkspaceEnabled    bool
	ForWorkspaceEnabled sql.NullBool
	Permissions         []string
}
