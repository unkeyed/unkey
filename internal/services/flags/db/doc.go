// Package db holds the read queries of the flags service. Definitions and
// overrides are written elsewhere, so this package has no write queries.
//
// The package uses sqlc to generate type-safe query methods from SQL files in
// the queries/ subdirectory. Run `go generate ./...` from this directory to
// regenerate query code.
package db
