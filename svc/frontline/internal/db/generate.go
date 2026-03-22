package db

// Generation pipeline for sqlc query code.
//
// Step 1: remove stale *_generated.go files so renamed or deleted queries
//         do not leave orphan code behind.
//go:generate rm -rf ./*_generated.go || true

// Step 2: run sqlc, which reads sqlc.json, parses queries/, and emits
//         typed Go code plus a scaffold db file named "deleteme.go".
//go:generate go tool sqlc generate

// Step 3: delete deleteme.go because database.go already provides the
//         [DBTX] interface, [Queries] struct, and [New] constructor that
//         sqlc would otherwise generate with no replica or metrics support.
//go:generate rm ./deleteme.go
