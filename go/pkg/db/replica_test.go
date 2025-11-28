package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/db/testdriver"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
)

// newWithDriver creates a database instance using a custom driver (for testing)
func newWithDriver(driverName, dsn string, logger logging.Logger) (*database, error) {
	var db *sql.DB
	var err error

	err = retry.New(
		retry.Attempts(3),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
	).Do(func() error {
		db, err = sql.Open(driverName, dsn)
		if err != nil {
			logger.Info("database not ready yet, retrying...", "error", err.Error())
		}
		return err
	})

	if err != nil {
		return nil, err
	}

	// Initialize replica
	replica := &Replica{
		db:         db,
		mode:       "rw",
		dsn:        dsn,
		logger:     logger,
		maxRetries: 3,
	}

	// Log hostname
	replica.logReplicaHostname(context.Background())

	return &database{
		writeReplica: replica,
		readReplica:  replica,
		logger:       logger,
	}, nil
}

func TestIsUnhealthyTabletError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "unhealthy tablet error",
			err:      errors.New("target: unkey.-.rdonly: no healthy tablet available for 'keyspace:\"unkey\" shard:\"-\" tablet_type:RDONLY'"),
			expected: true,
		},
		{
			name:     "partial unhealthy tablet error",
			err:      errors.New("connection failed: no healthy tablet available"),
			expected: true,
		},
		{
			name:     "different error",
			err:      errors.New("connection timeout"),
			expected: false,
		},
		{
			name:     "sql no rows error",
			err:      errors.New("sql: no rows in result set"),
			expected: false,
		},
		{
			name:     "empty error message",
			err:      errors.New(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnhealthyTabletError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestReplicaReconnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Register mock driver
	mockConfig := testdriver.Register()
	defer mockConfig.Reset()

	// Set up test database
	mysqlCfg := containers.MySQL(t)
	mysqlCfg.DBName = "unkey"
	dsn := mysqlCfg.FormatDSN()

	t.Run("reconnects on unhealthy tablet error", func(t *testing.T) {
		mockConfig.Reset()

		// Create database with mock driver
		db, err := newWithDriver("mysql-mock", dsn, logging.NewNoop())
		require.NoError(t, err)
		defer db.Close()

		// First connection should succeed
		initialAttempts := mockConfig.ConnectionAttempts()
		require.Equal(t, 1, initialAttempts, "should have 1 initial connection")

		// Make next query fail with unhealthy tablet error
		unhealthyErr := errors.New("no healthy tablet available for keyspace")
		mockConfig.FailNextQueries(1, unhealthyErr)

		// Trigger a query that will fail
		_, err = db.RW().QueryContext(ctx, "SELECT 1")

		// The query should fail
		require.Error(t, err)
		require.Contains(t, err.Error(), "no healthy tablet available")

		// Wait a bit for reconnection to happen in background
		time.Sleep(200 * time.Millisecond)

		// Should have attempted reconnection
		attempts := mockConfig.ConnectionAttempts()
		require.Greater(t, attempts, initialAttempts, "should have attempted reconnection")
	})

	t.Run("respects max retries", func(t *testing.T) {
		mockConfig.Reset()

		// Create database with mock driver
		db, err := newWithDriver("mysql-mock", dsn, logging.NewNoop())
		require.NoError(t, err)
		defer db.Close()

		initialAttempts := mockConfig.ConnectionAttempts()

		// Make all queries and reconnection attempts fail
		unhealthyErr := errors.New("no healthy tablet available for keyspace")
		mockConfig.FailNextQueries(20, unhealthyErr) // More than enough to trigger maxRetries
		mockConfig.FailNext(10, unhealthyErr)        // Make reconnections also fail

		// Trigger error multiple times
		for i := 0; i < 5; i++ {
			_, _ = db.RW().QueryContext(ctx, "SELECT 1")
			time.Sleep(50 * time.Millisecond)
		}

		// Wait for reconnections to complete
		time.Sleep(500 * time.Millisecond)

		attempts := mockConfig.ConnectionAttempts()
		// Should stop at maxRetries (3) + initial (1) = 4
		require.LessOrEqual(t, attempts, initialAttempts+4,
			"should respect max retries and not reconnect infinitely")
	})

	t.Run("stores and logs hostname", func(t *testing.T) {
		mockConfig.Reset()

		// Use real database for this test
		db, err := New(Config{
			PrimaryDSN: dsn, // Use real driver
			Logger:     logging.NewNoop(),
		})
		require.NoError(t, err)
		defer db.Close()

		// Hostname should be set
		replica := db.RW()
		require.NotEmpty(t, replica.hostname, "hostname should be stored")
	})
}
