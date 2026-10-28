package db

//go:generate sh -c "rm -f ./*_generated.go"
//go:generate go build -o ../../mysql/plugins/dist/bulk-insert ../../mysql/plugins/bulk-insert
//go:generate go tool sqlc generate
//go:generate rm ./delete_me.go
