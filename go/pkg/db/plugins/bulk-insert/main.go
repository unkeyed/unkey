// Package main provides a sqlc plugin that generates bulk insert functions.
//
// This plugin automatically creates bulk insert functions for existing insert queries,
// allowing efficient batch operations with a single SQL query instead of multiple
// individual INSERT statements.
package main

import (
	"context"

	"github.com/sqlc-dev/plugin-sdk-go/codegen"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// main is the entry point for the sqlc plugin.
func main() {
	codegen.Run(Generate)
}

// Generate processes the sqlc plugin request and generates bulk insert functions.
func Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	generator := NewGenerator()
	return generator.Generate(req)
}
