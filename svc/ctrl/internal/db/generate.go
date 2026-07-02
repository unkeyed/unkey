package db

//go:generate rm -rf ./*_generated.go || true
//go:generate go build -o ../../../../pkg/mysql/plugins/dist/bulk-insert ../../../../pkg/mysql/plugins/bulk-insert
//go:generate go tool sqlc generate
// We copy the relevant exports into queries.go and do not want sqlc's default
// database wrapper to be committed.
//go:generate rm delete_me.go
