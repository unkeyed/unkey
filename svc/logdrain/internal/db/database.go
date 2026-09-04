package db

import (
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

type database struct {
	primary *mysql.Replica
	*Queries
}

// New creates a log drain database from a single read-write MySQL DSN.
func New(dsn string, tags sqlcomment.Static) (*database, error) {
	primary, err := mysql.NewReplica(dsn, "rw", tags)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("cannot open log drain database"))
	}

	return &database{
		primary: primary,
		Queries: NewQueries(primary),
	}, nil
}

// Conn returns the single read-write connection used by log drain queries.
func (d *database) Conn() *Replica {
	return d.primary
}

// RW returns the single read-write connection.
func (d *database) RW() *Replica {
	return d.primary
}

// RO returns the single read-write connection.
func (d *database) RO() *Replica {
	return d.primary
}

// Close releases the log drain database connection pool.
func (d *database) Close() error {
	if err := d.primary.Close(); err != nil {
		return fault.Wrap(err)
	}
	return nil
}
