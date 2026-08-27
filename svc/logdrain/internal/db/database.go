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

func New(dsn string, tags sqlcomment.Static) (*database, error) {
	p, err := mysql.NewReplica(dsn, "rw", tags)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("cannot open logdrain database"))
	}
	return &database{primary: p, Queries: NewQueries(p)}, nil
}
func (d *database) Conn() *Replica { return d.primary }
func (d *database) Close() error   { return fault.Wrap(d.primary.Close()) }
func IsNotFound(err error) bool    { return mysql.IsNotFound(err) }
