package db

// Generation pipeline for sqlc query code.
//
// Step 1: remove stale *_generated.go files so renamed or deleted queries
//         do not leave orphan code behind.
//go:generate rm -rf ./*_generated.go || true

// Step 2: build the shared sqlc bulk-insert plugin used by sqlc.json.
//go:generate go build -o ../../../../pkg/mysql/plugins/dist/bulk-insert ../../../../pkg/mysql/plugins/bulk-insert

// Step 3: run sqlc, which reads sqlc.json, parses queries/, and emits
//         typed Go code plus a scaffold db file named "deleteme.go".
//go:generate go tool sqlc generate

// Step 4: delete deleteme.go because database.go already provides the
//         [DBTX] interface, [Queries] struct, and [New] constructor with the
//         primary/replica split that sqlc would otherwise overwrite.
//go:generate rm ./deleteme.go
