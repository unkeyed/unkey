package db

// DBTX is defined by sqlc in the generated code (delete_me_generated.go is removed).
// We re-declare it here along with the Queries struct so that the generated query
// methods compile. The interface is structurally identical to mysql.DBTX, so
// *mysql.Replica satisfies it via Go's structural typing.

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/mysql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Queries struct{}

// Query provides access to the generated database queries.
var Query Querier = &Queries{}

type Database = mysql.Database
