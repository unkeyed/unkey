package database

import (
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

type Database = gen.Querier

func New(dsn string) (Database, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("unable to open mysql connection", ""))
	}

	return gen.New(db), nil

}
