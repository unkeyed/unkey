package db

import "github.com/unkeyed/unkey/pkg/mysql"

type Queries struct {
	db mysql.DBTX
}

func NewQueries(db mysql.DBTX) *Queries {
	return &Queries{db: db}
}
