package store

// Generation pipeline for sqlc query code.
//
// Remove stale generated files so renamed or deleted queries do not leave
// orphan code behind, then run sqlc, which reads sqlc.json, parses queries/
// against the migrations as schema, and emits typed Go code.
//go:generate rm -f ./*_generated.go
//go:generate go tool sqlc generate
