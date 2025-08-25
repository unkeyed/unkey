package db

//go:generate sqlc generate
// we copy all of the relevant bits into query.go and don't want the default
// exports that get generated
//go:generate rm delete_me.go
