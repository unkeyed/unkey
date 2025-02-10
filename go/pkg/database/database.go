package database

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Config struct {
	// The primary DSN for your database. This must support both reads and writes.
	PrimaryDSN string

	// The readonly replica will be used for most read queries.
	// If omitted, the primary is used.
	ReadOnlyDSN string

	Logger logging.Logger
}

type replica struct {
	db    *sql.DB
	query *gen.Queries
}

type database struct {
	writeReplica *replica
	readReplica  *replica
	logger       logging.Logger
}

func New(config Config, middlewares ...Middleware) (Database, error) {

	write, err := sql.Open("mysql", config.PrimaryDSN)
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("cannot open primary replica", ""))
	}

	writeReplica := &replica{
		db:    write,
		query: gen.New(write),
	}
	readReplica := &replica{
		db:    write,
		query: gen.New(write),
	}
	if config.ReadOnlyDSN != "" {
		read, err := sql.Open("mysql", config.ReadOnlyDSN)
		if err != nil {
			return nil, fault.Wrap(err, fault.WithDesc("cannot open read replica", ""))
		}
		readReplica = &replica{
			db:    read,
			query: gen.New(read),
		}

	}

	var wrapped Database = &database{
		writeReplica: writeReplica,
		readReplica:  readReplica,
		logger:       config.Logger,
	}

	for _, mw := range middlewares {
		wrapped = mw(wrapped)
	}

	return wrapped, nil

}

func (d *database) write() *gen.Queries {
	return d.writeReplica.query
}

func (d *database) read() *gen.Queries {
	if d.readReplica != nil {
		return d.readReplica.query
	}
	return d.writeReplica.query
}

func (d *database) Close() error {
	writeCloseErr := d.writeReplica.db.Close()

	if d.readReplica != nil {
		readCloseErr := d.readReplica.db.Close()
		if readCloseErr != nil {
			return fault.Wrap(readCloseErr)
		}
	}
	if writeCloseErr != nil {
		return fault.Wrap(writeCloseErr)
	}
	return nil

}
