package database

import (
	_ "embed"
)

// Schema is the sql schema embedded into the binary
//
//go:embed schema.sql
var Schema []byte
