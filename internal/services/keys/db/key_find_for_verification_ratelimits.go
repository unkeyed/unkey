package db

// KeyFindForVerificationRatelimit represents a rate limit configuration extracted
// from the JSON column in the FindKeyForVerification query result. It covers both
// key-level and identity-level rate limits.
type KeyFindForVerificationRatelimit struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Limit      int    `json:"limit"`
	Duration   int    `json:"duration"`
	AutoApply  int    `json:"auto_apply"`
	KeyID      string `json:"key_id"`
	IdentityID string `json:"identity_id"`
}
