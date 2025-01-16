package database

import (
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
)

type Database = gen.Querier

func New(dsn string) (Database, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return gen.New(db), nil

}
