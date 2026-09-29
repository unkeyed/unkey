package caches

// RootKeyCacheKey separates root authentication from regular API-key verification.
// Invalidating a legacy key must remove both its raw hash and this entry.
func RootKeyCacheKey(hash string) string {
	return "root:" + hash
}
