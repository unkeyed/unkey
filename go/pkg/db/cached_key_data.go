package db

// CachedKeyData embeds FindKeyForVerificationRow and adds pre-processed data for caching.
// This struct is stored in the cache to avoid redundant parsing operations.
type CachedKeyData struct {
	FindKeyForVerificationRow
	ParsedIPWhitelist map[string]struct{} // Pre-parsed IP addresses for O(1) lookup
}
