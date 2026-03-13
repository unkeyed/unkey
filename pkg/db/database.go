package db

import (
	"context"
	"database/sql"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/retry"
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
}

// database implements the Database interface, providing access to database replicas
// and handling connection lifecycle.
type database struct {
	writeReplica *mysql.Replica // Primary database connection used for write operations
	readReplica  *mysql.Replica // Connection used for read operations (may be same as primary)
}

func open(dsn string) (db *sql.DB, err error) {
	if !strings.Contains(dsn, "parseTime=true") {
		return nil, fault.New("DSN must contain parseTime=true, see https://stackoverflow.com/questions/29341590/how-to-parse-time-from-database/29343013#29343013")
	}

	// sql.Open only validates the DSN, it doesn't actually connect.
	// We need to call Ping() to verify connectivity.
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to open database"))
	}

	// Configure connection pool for better performance
	// These settings prevent cold-start latency by maintaining warm connections
	db.SetMaxOpenConns(25)                 // Max concurrent connections
	db.SetMaxIdleConns(10)                 // Keep 10 idle connections ready
	db.SetConnMaxLifetime(5 * time.Minute) // Refresh connections every 5 min (PlanetScale recommendation)
	db.SetConnMaxIdleTime(1 * time.Minute) // Close idle connections after 1 min of inactivity

	// Verify connectivity at startup with retries - this establishes at least one connection
	// so the first request doesn't pay the connection establishment cost
	err = retry.New(
		retry.Attempts(5),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
	).Do(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		pingErr := db.PingContext(ctx)
		if pingErr != nil {
			logger.Info("mysql not ready yet, retrying...", "error", pingErr.Error())
		}
		return pingErr
	})
	if err != nil {
		_ = db.Close()
		return nil, fault.Wrap(err, fault.Internal("failed to ping database after retries"))
	}

	logger.Info("database connection pool initialized successfully")
	return db, nil
}

// New creates a new database instance with the provided configuration.
// It establishes connections to the primary database and optionally to a read-only replica.
// Returns an error if connections cannot be established or if DSNs are misconfigured.
func New(config Config) (mysql.Database, error) {

	return mysql.New(mysql.Config{
		PrimaryDSN:  config.PrimaryDSN,
		ReadOnlyDSN: config.ReadOnlyDSN,
	})
}
