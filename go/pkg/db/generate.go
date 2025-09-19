package db

//go:generate go build -o ./plugins/dist/bulk-insert ./plugins/bulk-insert
//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc generate
// we copy all of the relevant bits into query.go and don't want the default
// exports that get generated
//go:generate rm delete_me.go
