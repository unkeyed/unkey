package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type database struct {
	primary     *sql.DB
	readReplica *sql.DB
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

func New(config Config) (Database, error) {
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

	var readReplica *sql.DB = nil
	c := getClosestContinent(config.FlyRegion)
	if c == continentEu && config.ReplicaEu != "" {
		logger.Info("Adding database read replica", zap.String("continent", "europe"))

		readReplica, err = sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.ReplicaEu))
		if err != nil {
			return nil, fmt.Errorf("error opening database: %w", err)
		}
	} else if c == continentAsia && config.ReplicaAsia != "" {
		logger.Info("Adding database read replica", zap.String("continent", "asia"))

		readReplica, err = sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.ReplicaAsia))
		if err != nil {
			return nil, fmt.Errorf("error opening database: %w", err)
		}
	}

	if readReplica != nil {
		err = readReplica.Ping()
		if err != nil {
			return nil, fmt.Errorf("unable to ping read replica")
		}
		if config.PlanetscaleBoost {
			logger.Info("enabling planetscale boost for replica")

			_, err = readReplica.Exec("SET @@boost_cached_queries = true")
			if err != nil {
				return nil, fmt.Errorf("cannot enable bloost on replica: %w", err)
			}
		}
	}

	return &database{
		primary:     primary,
		readReplica: readReplica,
		logger:      logger,
	}, nil

}

// read returns the primary writable db
func (d *database) write() *sql.DB {
	return d.primary
}

// read returns the closests read replica
func (d *database) read() *sql.DB {
	if d.readReplica != nil {
		return d.readReplica
	}
	return d.primary
}
