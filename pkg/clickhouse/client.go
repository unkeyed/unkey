package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/unkeyed/unkey/pkg/assert"
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

// QueryResultLimits bounds dynamic query results while scanning them into memory.
// Both values must be positive and cannot exceed the analytics hard maximums.
type QueryResultLimits struct {
	RowsMax  int
	BytesMax int
}

const (
	// AnalyticsResultRowsMax is the maximum number of rows returned by customer analytics queries.
	AnalyticsResultRowsMax = 10_000
	// AnalyticsResultBytesMax is the maximum encoded size of customer analytics results.
	AnalyticsResultBytesMax = 4 << 20
	// AnalyticsASTDepthMax is the maximum ClickHouse AST depth for customer analytics queries.
	AnalyticsASTDepthMax = 100
	// AnalyticsASTElementsMax is the maximum number of ClickHouse AST elements for customer analytics queries.
	AnalyticsASTElementsMax = 2_000
)

// AnalyticsWorkspaceResultRowsMax returns the lower of a positive workspace
// limit and the API hard limit. Non-positive workspace values use the hard limit.
func AnalyticsWorkspaceResultRowsMax(workspaceRowsMax int32) int32 {
	if workspaceRowsMax > 0 && workspaceRowsMax < int32(AnalyticsResultRowsMax) {
		return workspaceRowsMax
	}
	return int32(AnalyticsResultRowsMax)
}

// validate rejects programmer errors that would disable a mandatory result bound.
func (limits QueryResultLimits) validate() error {
	return assert.All(
		assert.Greater(limits.RowsMax, 0, "query result row limit must be positive"),
		assert.LessOrEqual(limits.RowsMax, AnalyticsResultRowsMax, "query result row limit exceeds the hard maximum"),
		assert.Greater(limits.BytesMax, 0, "query result byte limit must be positive"),
		assert.LessOrEqual(limits.BytesMax, AnalyticsResultBytesMax, "query result byte limit exceeds the hard maximum"),
	)
}

// QueryToMaps executes a dynamic query and scans results within mandatory row
// and encoded-byte bounds. It cancels the query before closing partially read rows.
func (c *Client) QueryToMaps(ctx context.Context, query string, limits QueryResultLimits, arguments ...any) ([]map[string]any, error) {
	if err := limits.validate(); err != nil {
		return nil, err
	}

	ctxQuery, cancel := context.WithCancel(ctx)
	defer cancel()
	rows, err := c.conn.Query(ctxQuery, query, arguments...)
	if err != nil {
		return nil, WrapClickHouseError(err)
	}

	results, errScan := queryToMapsScanRows(rows, limits)
	if errScan != nil {
		cancel()
	}
	errClose := rows.Close()
	if errScan != nil {
		return nil, errScan
	}
	if errClose != nil {
		return nil, WrapClickHouseError(errClose)
	}
	return results, nil
}

// QueryToMaps scans dynamic rows while accounting for their JSON encoding.
func queryToMapsScanRows(rows driver.Rows, limits QueryResultLimits) ([]map[string]any, error) {
	columns := rows.Columns()
	results := make([]map[string]any, 0, limits.RowsMax)
	resultSizeBytes := 2 // JSON array brackets.

	for rows.Next() {
		if len(results) >= limits.RowsMax {
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
			return nil, fault.Wrap(err, fault.Public("Failed to read query results"))
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		rowBytes, errMarshal := json.Marshal(row)
		if errMarshal != nil {
			return nil, fault.Wrap(errMarshal, fault.Public("Failed to encode query results"))
		}
		resultSizeBytes += len(rowBytes)
		if len(results) > 0 {
			resultSizeBytes++
		}
		if resultSizeBytes > limits.BytesMax {
			return nil, fault.New("result byte limit exceeded",
				fault.Code(codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN()),
				fault.Public("Query result exceeds the maximum response size."),
			)
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
