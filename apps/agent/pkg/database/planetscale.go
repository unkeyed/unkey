package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"go.uber.org/zap"
)

type replica struct {
	db    *sql.DB
	query *gen.Queries
}

type database struct {
	writeReplica *replica
	readReplica  *replica
	logger       *zap.Logger
}

type Config struct {
	PrimaryUs   string
	ReplicaEu   string
	ReplicaAsia string
	FlyRegion   string
	Logger      *zap.Logger
}

type Middleware func(Database) Database

func New(config Config, middlewares ...Middleware) (Database, error) {
	logger := config.Logger.With(zap.String("pkg", "database"))
	primary, err := connect(config.PrimaryUs)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to primary, %w", err)
	}
	var readDB *sql.DB
	c := getClosestContinent(config.FlyRegion)
	if c == continentEu && config.ReplicaEu != "" {
		logger.Info("Adding database read replica", zap.String("continent", "europe"))
		readDB, err = connect(config.ReplicaEu)
		if err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to connect to europe replica")
			}
		}
	} else if c == continentAsia && config.ReplicaAsia != "" {
		logger.Info("Adding database read replica", zap.String("continent", "asia"))
		readDB, err = connect(config.ReplicaAsia)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to asia replica")
		}
	} else {
		logger.Info("Adding database read replica", zap.String("continent", "us"))
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
	d.logger.Warn("falling back to primary ")
	return d.writeReplica.query
}

func (d *database) Close() error {
	err := d.readReplica.db.Close()
	if err != nil {
		return err
	}
	return d.readReplica.db.Close()
}
