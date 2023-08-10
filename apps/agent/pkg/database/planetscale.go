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
	primary     *replica
	readReplica *replica
	logger      *zap.Logger
}

type Config struct {
	PrimaryUs        string
	ReplicaEu        string
	ReplicaAsia      string
	FlyRegion        string
	Logger           *zap.Logger
	PlanetscaleBoost bool
}

type Middleware func(Database) Database

func New(config Config, middlewares ...Middleware) (Database, error) {
	logger := config.Logger.With(zap.String("pkg", "database"))
	primary, err := sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.PrimaryUs))
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	if config.PlanetscaleBoost {
		logger.Info("enabling planetscale boost for primary")
		_, err = primary.Exec("SET @@boost_cached_queries = true")
		if err != nil {
			return nil, fmt.Errorf("cannot enable bloost: %w", err)
		}
	}

	err = primary.Ping()
	if err != nil {
		return nil, fmt.Errorf("unable to ping database")
	}

	var readDB *sql.DB = nil
	c := getClosestContinent(config.FlyRegion)
	if c == continentEu && config.ReplicaEu != "" {
		logger.Info("Adding database read replica", zap.String("continent", "europe"))

		readDB, err = sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.ReplicaEu))
		if err != nil {
			return nil, fmt.Errorf("error opening database: %w", err)
		}
	} else if c == continentAsia && config.ReplicaAsia != "" {
		logger.Info("Adding database read replica", zap.String("continent", "asia"))

		readDB, err = sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.ReplicaAsia))
		if err != nil {
			return nil, fmt.Errorf("error opening database: %w", err)
		}
	}

	if readDB != nil {
		err = readDB.Ping()
		if err != nil {
			return nil, fmt.Errorf("unable to ping read replica")
		}
		if config.PlanetscaleBoost {
			logger.Info("enabling planetscale boost for replica")

			_, err = readDB.Exec("SET @@boost_cached_queries = true")
			if err != nil {
				return nil, fmt.Errorf("cannot enable bloost on replica: %w", err)
			}
		}
	}

	primaryReplica := &replica{
		db:    primary,
		query: gen.New(primary),
	}
	var readReplica *replica
	if readDB != nil {
		readReplica = &replica{
			db:    readDB,
			query: gen.New(readDB),
		}
	}

	var db Database = &database{
		primary:     primaryReplica,
		readReplica: readReplica,
		logger:      logger,
	}
	for _, mw := range middlewares {
		db = mw(db)
	}
	return db, nil
}

// read returns the primary writable db
func (d *database) write() *gen.Queries {
	return d.primary.query
}

// read returns the closests read replica or primary as fallback
func (d *database) read() *gen.Queries {
	if d.readReplica != nil && d.readReplica.query != nil {
		return d.readReplica.query
	}
	if d.primary != nil && d.primary.query != nil {
		return d.primary.query
	}

	panic("neither primary, nor read replica are available")
}
