package db

import "github.com/unkeyed/unkey/pkg/mysql"

type Replica = mysql.Replica
type DBTX = mysql.DBTX
type DBTx = mysql.DBTx
type Database interface {
	Querier
	Conn() *Replica
	Close() error
}
