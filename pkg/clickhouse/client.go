package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/retry"
)

// Client represents a client for interacting with a ClickHouse database.
// Batch processing for different event types is handled externally via
// NewBuffer[T], which wires a *batch.BatchProcessor to this client's
// connection, retry policy, and circuit breaker.
type Client struct {
	conn           ch.Conn
	circuitBreaker *circuitbreaker.CB[struct{}]
	retry          *retry.Retry
}

var (
	_ Querier    = (*Client)(nil)
	_ ClickHouse = (*Client)(nil)
)

// Config contains the configuration options for the ClickHouse client.
type Config struct {
	// URL is the ClickHouse connection string
	// Format: clickhouse://username:password@host:port/database?param1=value1&...
	URL string
}

// New creates a new ClickHouse client with the provided configuration.
// It establishes a connection to the ClickHouse server but does not create
// any batch processors. Use NewBuffer[T] to create type-safe batch processors
// for specific event types.
//
// Example:
//
//	client, err := clickhouse.New(clickhouse.Config{
//	    URL: "clickhouse://user:pass@clickhouse.example.com:9000/db",
//	})
//	if err != nil {
//	    return fmt.Errorf("failed to initialize clickhouse: %w", err)
//	}
//	buf := clickhouse.NewBuffer[schema.ApiRequest](client, "default.api_requests_raw_v2", clickhouse.BufferConfig{...})
func New(config Config) (*Client, error) {
	opts, err := ch.ParseDSN(config.URL)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("parsing clickhouse DSN failed"))
	}

	logger.Info("initializing clickhouse client")
	opts.Debug = true
	opts.Debugf = func(format string, v ...any) {
		logger.Debug(fmt.Sprintf(format, v...))
	}
	opts.MaxOpenConns = 50
	opts.ConnMaxLifetime = time.Hour
	opts.ConnOpenStrategy = ch.ConnOpenRoundRobin
	opts.DialTimeout = 5 * time.Second // Fail fast on connection issues

	logger.Info("connecting to clickhouse")
	conn, err := ch.Open(opts)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("opening clickhouse failed"))
	}

	err = retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
		retry.ShouldRetry(func(err error) bool {
			// Don't retry authentication errors - they won't succeed without credential changes
			return !isAuthenticationError(err)
		}),
	).
		Do(func() error {
			return conn.Ping(context.Background())
		})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("pinging clickhouse failed"))
	}

	c := &Client{
		conn: conn,
		circuitBreaker: circuitbreaker.New[struct{}](
			"clickhouse_insert",
			circuitbreaker.WithTripThreshold(5),
			circuitbreaker.WithTimeout(30*time.Second),
			circuitbreaker.WithCyclicPeriod(10*time.Second),
			circuitbreaker.WithMaxRequests(3),
		),
		retry: retry.New(
			retry.Attempts(5),
			retry.Backoff(func(n int) time.Duration {
				return time.Duration(1<<uint(n-1)) * time.Second
			}),
			retry.ShouldRetry(func(err error) bool {
				return !isAuthenticationError(err)
			}),
		),
	}

	return c, nil
}

// isAuthenticationError checks if an error is related to authentication/authorization
// These errors should not be retried as they won't succeed without credential changes
func isAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	// ClickHouse authentication/authorization error patterns
	return strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "password") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "code: 516") || // Authentication failed
		strings.Contains(errStr, "code: 517") // Wrong password
}

func (c *Client) Conn() ch.Conn {
	return c.conn
}

// QueryToMaps executes a query and scans all rows into a slice of maps.
// Each map represents a row with column names as keys and values as ch.Dynamic.
// Returns fault-wrapped errors with appropriate codes for resource limits,
// user query errors, and system errors.
func (c *Client) QueryToMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, WrapClickHouseError(err)
	}
	defer func() { _ = rows.Close() }()

	columns := rows.Columns()
	results := make([]map[string]any, 0)

	for rows.Next() {
		// Create slice of ch.Dynamic to scan into
		values := make([]ch.Dynamic, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fault.Wrap(err, fault.Public("Failed to read query results"))
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, WrapClickHouseError(err)
	}

	return results, nil
}

// Exec executes a DDL or DML statement that doesn't return rows.
// Used for CREATE, ALTER, DROP, GRANT, REVOKE, etc.
func (c *Client) Exec(ctx context.Context, sql string, args ...any) error {
	return c.conn.Exec(ctx, sql, args...)
}

func (c *Client) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close shuts down the ClickHouse connection.
// Any batch processors created via NewBuffer must be closed separately
// (and before this call) to ensure buffered rows are flushed.
func (c *Client) Close() error {
	err := c.conn.Close()
	if err != nil {
		return fault.Wrap(err, fault.Internal("clickhouse couldn't shut down"))
	}

	return nil
}
