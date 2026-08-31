package db

import (
	"database/sql"
)

type FindRatelimitNamespaceLimitOverride struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Limit      int64  `json:"limit"`
	Duration   int64  `json:"duration"`
}

type FindRatelimitNamespace struct {
	ID                string                                         `db:"id"`
	WorkspaceID       string                                         `db:"workspace_id"`
	Name              string                                         `db:"name"`
	CreatedAtM        int64                                          `db:"created_at_m"`
	UpdatedAtM        sql.NullInt64                                  `db:"updated_at_m"`
	DeletedAtM        sql.NullInt64                                  `db:"deleted_at_m"`
	DirectOverrides   map[string]FindRatelimitNamespaceLimitOverride `db:"direct_overrides"`
	WildcardOverrides []FindRatelimitNamespaceLimitOverride          `db:"wildcard_overrides"`
}
