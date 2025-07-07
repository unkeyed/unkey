package db

type KeyFindForVerificationRatelimit struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	KeyID      string `db:"key_id"`
	IdentityID string `db:"identity_id"`
	Limit      int    `db:"limit"`
	Duration   int    `db:"duration"`
	AutoApply  bool   `db:"auto_apply"`
}
