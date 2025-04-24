package schema

import (
	"embed"
)

// Migrations are the raw sql files for the tables.
//
//go:embed databases/**/*.sql
var Migrations embed.FS
