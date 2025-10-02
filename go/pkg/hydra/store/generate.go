package store

//go:generate sqlc generate -f sqlc.json
// we copy all of the relevant bits into queries.go and don't want the default
// exports that get generated
//go:generate rm delete_me.go
