package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/codes"
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
//	buf := clickhouse.NewBuffer[schema.ApiRequest](client, clickhouse.BufferConfig{...})
func New(config Config) (*Client, error) {
	opts, err := ch.ParseDSN(config.URL)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("parsing clickhouse DSN failed"))
	}

	logger.Info("initializing clickhouse client")
	// The clickhouse-go driver emits a handful of messages per query at
	// Debug: send data, table columns, profile events, released, etc. Routing
	// those through our normal logger.Debug drowns out every other debug line
	// when UNKEY_LOG_LEVEL=debug is flipped on to investigate something else
	// (heimdall attach, collector tick, etc.). Gate the driver's debug spam
	// behind a dedicated env var so it stays opt-in for the rare case where
	// you are actually debugging the driver itself.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("UNKEY_CLICKHOUSE_DRIVER_DEBUG")), "true") {
		opts.Debug = true
		opts.Debugf = func(format string, v ...any) {
			logger.Debug(fmt.Sprintf(format, v...))
		}
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
	return c.queryToMaps(ctx, query, ResultLimits{MaxRows: 0, MaxBytes: 0}, func() {}, args...)
}

// ResultLimits bounds dynamic analytics results both at ClickHouse and while
// scanning into application memory. Zero values disable the corresponding cap.
type ResultLimits struct {
	MaxRows  int
	MaxBytes int
}

const (
	AnalyticsMaxResultRows  = 10_000
	AnalyticsMaxResultBytes = 4 << 20
	AnalyticsMaxASTDepth    = 100
	AnalyticsMaxASTElements = 2_000
)

func (c *Client) QueryToMapsBounded(ctx context.Context, query string, limits ResultLimits, args ...any) ([]map[string]any, error) {
	queryCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	return c.queryToMaps(queryCtx, query, limits, cancel, args...)
}

func (c *Client) queryToMaps(ctx context.Context, query string, limits ResultLimits, cancel context.CancelFunc, args ...any) ([]map[string]any, error) {
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, WrapClickHouseError(err)
	}
	defer func() { _ = rows.Close() }()

	columns := rows.Columns()
	capacity := limits.MaxRows
	if capacity <= 0 {
		capacity = 0
	}
	results := make([]map[string]any, 0, capacity)
	encodedBytes := 2 // JSON array brackets

	for rows.Next() {
		if limits.MaxRows > 0 && len(results) >= limits.MaxRows {
			cancel()
			return nil, fault.New("result row limit exceeded",
				fault.Code(codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN()),
				fault.Public("Query result exceeds the maximum row count."),
			)
		}

		// Create slice of ch.Dynamic to scan into
		values := make([]ch.Dynamic, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			cancel()
			return nil, fault.Wrap(err, fault.Public("Failed to read query results"))
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		if limits.MaxBytes > 0 {
			encodedRow, marshalErr := json.Marshal(row)
			if marshalErr != nil {
				cancel()
				return nil, fault.Wrap(marshalErr, fault.Public("Failed to encode query results"))
			}
			encodedBytes += len(encodedRow)
			if len(results) > 0 {
				encodedBytes++
			}
			if encodedBytes > limits.MaxBytes {
				cancel()
				return nil, fault.New("result byte limit exceeded",
					fault.Code(codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN()),
					fault.Public("Query result exceeds the maximum response size."),
				)
			}
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
