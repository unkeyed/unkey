package db

import (
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
)

type RoleInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description dbtype.NullString `json:"description"`
}

type PermissionInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description dbtype.NullString `json:"description"`
}

type RatelimitInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	KeyID      dbtype.NullString `json:"key_id"`
	IdentityID dbtype.NullString `json:"identity_id"`
	Limit      int32             `json:"limit"`
	Duration   int64             `json:"duration"`
	AutoApply  bool              `json:"auto_apply"`
}
