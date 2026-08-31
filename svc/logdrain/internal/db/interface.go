package db

import "github.com/unkeyed/unkey/pkg/mysql"

// Replica is a traced MySQL connection pool.
type Replica = mysql.Replica

// DBTX is the interface accepted by generated log drain queries.
type DBTX = mysql.DBTX

// DBTx is a transactional MySQL connection.
type DBTx = mysql.DBTx

// Database defines the single read-write database used by the log drain service.
type Database interface {
	Querier

	// Conn returns the single read-write connection.
	Conn() *Replica

	// RW returns the single read-write connection.
	RW() *Replica

	// RO returns the single read-write connection.
	RO() *Replica

	// Close terminates the database connection.
	Close() error
}
