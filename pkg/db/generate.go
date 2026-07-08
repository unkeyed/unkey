package db

//go:generate go build -o ../mysql/plugins/dist/bulk-insert ../mysql/plugins/bulk-insert
//go:generate go tool sqlc generate
// we copy all of the relevant bits into query.go and don't want the default
// exports that get generated
//go:generate rm delete_me.go
