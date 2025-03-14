package db

import (
	"database/sql"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Config defines the parameters needed to establish database connections.
// It supports separate connections for read and write operations to allow
// for primary/replica setups.
type Config struct {
	// The primary DSN for your database. This must support both reads and writes.
	PrimaryDSN string

	// The readonly replica will be used for most read queries.
	// If omitted, the primary is used.
	ReadOnlyDSN string

	// Logger for database-related operations
	Logger logging.Logger
}

// database implements the Database interface, providing access to database replicas
// and handling connection lifecycle.
type database struct {
	writeReplica *Replica       // Primary database connection used for write operations
	readReplica  *Replica       // Connection used for read operations (may be same as primary)
	logger       logging.Logger // Logger for database operations
}

// New creates a new database instance with the provided configuration.
// It establishes connections to the primary database and optionally to a read-only replica.
// Returns an error if connections cannot be established or if DSNs are misconfigured.
func New(config Config) (*database, error) {
	// Validate DSN configuration
	if !strings.Contains(config.PrimaryDSN, "parseTime=true") {
		return nil, fault.New("PrimaryDSN must contain parseTime=true, see https://stackoverflow.com/questions/29341590/how-to-parse-time-from-database/29343013#29343013")
	}

	// Open primary database connection
	write, err := sql.Open("mysql", config.PrimaryDSN)
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("cannot open primary replica", ""))
	}

	// Initialize primary replica
	writeReplica := &Replica{
		db: write,
	}

	// Initialize read replica with primary by default
	readReplica := &Replica{
		db: write,
	}

	// If a separate read-only DSN is provided, establish that connection
	if config.ReadOnlyDSN != "" {
		if !strings.Contains(config.ReadOnlyDSN, "parseTime=true") {
			return nil, fault.New("ReadOnlyDSN must contain parseTime=true, see https://stackoverflow.com/questions/29341590/how-to-parse-time-from-database/29343013#29343013")
		}
		read, err := sql.Open("mysql", config.ReadOnlyDSN)
		if err != nil {
			return nil, fault.Wrap(err, fault.WithDesc("cannot open read replica", ""))
		}
		readReplica = &Replica{
			db: read,
		}
	}

	return &database{
		writeReplica: writeReplica,
		readReplica:  readReplica,
		logger:       config.Logger,
	}, nil
}

// RW returns the write replica for performing database write operations.
func (d *database) RW() *Replica {
	return d.writeReplica
}

// RO returns the read replica for performing database read operations.
// If no dedicated read replica is configured, it returns the write replica.
func (d *database) RO() *Replica {
	if d.readReplica != nil {
		return d.readReplica
	}
	return d.writeReplica
}

// Close properly closes all database connections.
// This should be called when the application is shutting down.
func (d *database) Close() error {
	// Close the write replica connection
	writeCloseErr := d.writeReplica.db.Close()

	// Only close the read replica if it's a separate connection
	if d.readReplica != nil {
		readCloseErr := d.readReplica.db.Close()
		if readCloseErr != nil {
			return fault.Wrap(readCloseErr)
		}
	}

	// Return any write replica close error
	if writeCloseErr != nil {
		return fault.Wrap(writeCloseErr)
	}
	return nil
}
