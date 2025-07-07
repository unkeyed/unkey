package db

import "database/sql"

type KeyFindForVerificationRatelimit struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Limit      int            `json:"limit"`
	Duration   int            `json:"duration"`
	AutoApply  bool           `json:"auto_apply"`
	KeyID      sql.NullString `json:"key_id"`
	IdentityID sql.NullString `json:"identity_id"`
}
