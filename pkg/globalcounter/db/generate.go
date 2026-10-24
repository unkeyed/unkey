package db

// sqlc infers Read's aggregate as nullable "any". The package-local override
// scans it into int64 so out-of-range sums fail rather than truncate in SQL.
//go:generate sh -c "rm -f ./*_generated.go"
//go:generate go build -o ../../mysql/plugins/dist/bulk-insert ../../mysql/plugins/bulk-insert
//go:generate go tool sqlc generate
//go:generate rm ./deleteme.go
