package db

type KeyFindForVerificationRatelimit struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Limit      int    `json:"limit"`
	Duration   int    `json:"duration"`
	AutoApply  int    `json:"auto_apply"`
	KeyID      string `json:"key_id"`
	IdentityID string `json:"identity_id"`
}
