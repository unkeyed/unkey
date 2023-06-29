package database

import (
	"database/sql"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type Database struct {
	primary     *sql.DB
	readReplica *sql.DB
	logger      logging.Logger
	tracer      tracing.Tracer
}

type Config struct {
	PrimaryUs   string
	ReplicaEu   string
	ReplicaAsia string
	FlyRegion   string
	Logger      logging.Logger
	Tracer      tracing.Tracer
}

func New(config Config) (*Database, error) {
	primary, err := sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.PrimaryUs))
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	err = primary.Ping()
	if err != nil {
		return nil, fmt.Errorf("unable to ping database")
	}

	var readReplica *sql.DB = nil
	c := getClosestContinent(config.FlyRegion)
	if c == continentEu && config.ReplicaEu != "" {
		config.Logger.Info("Adding database read replica", zap.String("continent", "europe"))

		readReplica, err = sql.Open("mysql", fmt.Sprintf("%s&parseTime=true", config.ReplicaEu))
		if err != nil {
			return nil, fmt.Errorf("error opening database: %w", err)
		}
	} else if c == continentAsia && config.ReplicaAsia != "" {
		config.Logger.Info("Adding database read replica", zap.String("continent", "asia"))

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
	}

	return &Database{
		primary:     primary,
		readReplica: readReplica,
		logger:      config.Logger,
		tracer:      config.Tracer,
	}, nil

}

// read returns the primary writable db
func (d *Database) write() *sql.DB {
	return d.primary
}

// read returns the closests read replica
func (d *Database) read() *sql.DB {
	if d.readReplica != nil {
		return d.readReplica
	}
	return d.primary
}
