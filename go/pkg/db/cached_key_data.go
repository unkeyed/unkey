package db

// CachedKeyData wraps FindKeyForVerificationRow with pre-processed data for caching.
// This struct is stored in the cache to avoid redundant parsing operations.
type CachedKeyData struct {
	Row               FindKeyForVerificationRow
	ParsedIPWhitelist []string // Pre-parsed and trimmed IP addresses from the whitelist
}
