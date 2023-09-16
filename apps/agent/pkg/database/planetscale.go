package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type replica struct {
	db    *sql.DB
	query *gen.Queries
}

type database struct {
	writeReplica *replica
	readReplica  *replica
	logger       logging.Logger
}

type Config struct {
	PrimaryUs   string
	ReplicaEu   string
	ReplicaAsia string
	FlyRegion   string
	Logger      logging.Logger
}

type Middleware func(Database) Database

func New(config Config, middlewares ...Middleware) (Database, error) {
	logger := config.Logger.With().Str("pkg", "database").Logger()
	primary, err := connect(config.PrimaryUs)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to primary, %w", err)
	}
	var readDB *sql.DB
	c := getClosestContinent(config.FlyRegion)
	if c == continentEu && config.ReplicaEu != "" {
		logger.Info().Str("continent", "europe").Msg("Adding database read replica")
		readDB, err = connect(config.ReplicaEu)
		if err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to connect to europe replica")
			}
		}
	} else if c == continentAsia && config.ReplicaAsia != "" {
		logger.Info().Str("continent", "asia").Msg("Adding database read replica")
		readDB, err = connect(config.ReplicaAsia)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to asia replica")
		}
	} else {
		logger.Info().Str("continent", "us").Msg("Adding database read replica")
		readDB, err = connect(config.PrimaryUs)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to us replica")
		}
	}
	var db Database = &database{
		writeReplica: &replica{
			db:    primary,
			query: gen.New(primary),
		},
		readReplica: &replica{
			db:    readDB,
			query: gen.New(readDB),
		},
		logger: logger,
	}
	for _, mw := range middlewares {
		db = mw(db)
	}
	return db, nil
}

// read returns the primary writable db
func (d *database) write() *gen.Queries {
	return d.writeReplica.query
}

// read returns the closests read replica or primary as fallback
func (d *database) read() *gen.Queries {
	if d.readReplica != nil && d.readReplica.query != nil {
		return d.readReplica.query
	}
	d.logger.Warn().Msg("falling back to primary ")
	return d.writeReplica.query
}

func (d *database) Close() error {
	err := d.readReplica.db.Close()
	if err != nil {
		return err
	}
	return d.readReplica.db.Close()
}
