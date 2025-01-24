//go:build tools
// +build tools

package main

import (
	// oapi-codegen generates request and response body structs from openapi.json
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"

	// sqlc generates go code from raw sql queries
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
)
