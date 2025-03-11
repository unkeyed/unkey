package db

import (
	_ "embed"
)

// Schema is the SQL schema embedded into the binary.
// This allows the application to carry its own schema definition,
// which can be useful for initialization, validation, or migrations.
//
//go:embed schema.sql
var Schema []byte
