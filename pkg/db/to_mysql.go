package db

import "github.com/unkeyed/unkey/pkg/mysql"

// ToMySQL converts a [Database] into a [mysql.MySQL] by wrapping the
// underlying *sql.DB connections as mysql replicas. This is a temporary
// bridge while callers migrate from pkg/db to pkg/mysql. Remove this
// alongside mysql.NewReplicaFromDB and mysql.NewFromReplicas once all
// callers create databases via pkg/mysql.New directly.
func ToMySQL(d Database) mysql.MySQL {
	ro := d.RO()
	rw := d.RW()
	return mysql.NewFromReplicas(
		mysql.NewReplicaFromDB(ro.db, ro.mode),
		mysql.NewReplicaFromDB(rw.db, rw.mode),
	)
}
