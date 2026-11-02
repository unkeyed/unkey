package db

//go:generate sh -c "rm -f ./*_generated.go"
//go:generate go tool sqlc generate
//go:generate rm delete_me.go
