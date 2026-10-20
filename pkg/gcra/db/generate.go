package db

// sqlc infers the aggregates of ReadPeers as nullable "any". The package-local
// override scans them into int64, so a sum that does not fit fails instead of
// being truncated.
//go:generate sh -c "rm -f ./*_generated.go"
//go:generate go build -o ../../mysql/plugins/dist/bulk-insert ../../mysql/plugins/bulk-insert
//go:generate go tool sqlc generate
//go:generate rm ./deleteme.go
