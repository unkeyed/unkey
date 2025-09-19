package store

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc generate -f sqlc.json
// we copy all of the relevant bits into queries.go and don't want the default
// exports that get generated
//go:generate rm delete_me.go
