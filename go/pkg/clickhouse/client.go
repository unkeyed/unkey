package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/go/pkg/batch"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// Clickhouse represents a client for interacting with a ClickHouse database.
// It provides batch processing for different event types to efficiently store
// high volumes of data while minimizing connection overhead.
type clickhouse struct {
	conn   ch.Conn
	logger logging.Logger

	// Batched processors for different event types
	requests           *batch.BatchProcessor[schema.ApiRequestV1]
	apiRequests        *batch.BatchProcessor[schema.ApiRequestV2]
	keyVerifications   *batch.BatchProcessor[schema.KeyVerificationRequestV1]
	keyVerificationsV2 *batch.BatchProcessor[schema.KeyVerificationV2]
	ratelimits         *batch.BatchProcessor[schema.RatelimitRequestV1]
}

var _ Bufferer = (*clickhouse)(nil)
var _ Querier = (*clickhouse)(nil)
var _ ClickHouse = (*clickhouse)(nil)

// Config contains the configuration options for the ClickHouse client.
type Config struct {
	// URL is the ClickHouse connection string
	// Format: clickhouse://username:password@host:port/database?param1=value1&...
	URL string

	// Logger for ClickHouse operations
	Logger logging.Logger
}

// New creates a new ClickHouse client with the provided configuration.
// It establishes a connection to the ClickHouse server and initializes
// batch processors for different event types.
//
// The client uses batch processing to efficiently handle high volumes
// of events, automatically flushing based on batch size and time interval.
//
// Example:
//
//	ch, err := clickhouse.New(clickhouse.Config{
//	    URL:    "clickhouse://user:pass@clickhouse.example.com:9000/db",
//	    Logger: logger,
//	})
//	if err != nil {
//	    return fmt.Errorf("failed to initialize clickhouse: %w", err)
//	}
func New(config Config) (*clickhouse, error) {
	opts, err := ch.ParseDSN(config.URL)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("parsing clickhouse DSN failed"))
	}

	config.Logger.Info("initializing clickhouse client")
	opts.Debug = true
	opts.Debugf = func(format string, v ...any) {
		config.Logger.Debug(fmt.Sprintf(format, v...))
	}
	opts.MaxOpenConns = 50
	opts.ConnMaxLifetime = time.Hour
	opts.ConnOpenStrategy = ch.ConnOpenRoundRobin
	opts.DialTimeout = 5 * time.Second // Fail fast on connection issues

	config.Logger.Info("connecting to clickhouse")
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

	c := &clickhouse{
		conn:   conn,
		logger: config.Logger,

		requests: batch.New(batch.Config[schema.ApiRequestV1]{
			Name:          "requests",
			Drop:          true,
			BatchSize:     50_000,
			BufferSize:    200_000,
			FlushInterval: 5 * time.Second,
			Consumers:     2,
			Flush: func(ctx context.Context, rows []schema.ApiRequestV1) {
				table := "metrics.raw_api_requests_v1"
				err := flush(ctx, conn, table, rows)
				if err != nil {
					config.Logger.Error("failed to flush batch",
						"table", table,
						"err", err.Error(),
					)
				}
			},
		}),

		apiRequests: batch.New(batch.Config[schema.ApiRequestV2]{
			Name:          "api_requests",
			Drop:          true,
			BatchSize:     50_000,
			BufferSize:    200_000,
			FlushInterval: 5 * time.Second,
			Consumers:     2,
			Flush: func(ctx context.Context, rows []schema.ApiRequestV2) {
				table := "default.api_requests_raw_v2"
				err := flush(ctx, conn, table, rows)
				if err != nil {
					config.Logger.Error("failed to flush batch",
						"table", table,
						"err", err.Error(),
					)
				}
			},
		}),

		keyVerifications: batch.New[schema.KeyVerificationRequestV1](
			batch.Config[schema.KeyVerificationRequestV1]{
				Name:          "key_verifications",
				Drop:          true,
				BatchSize:     50_000,
				BufferSize:    200_000,
				FlushInterval: 5 * time.Second,
				Consumers:     2,
				Flush: func(ctx context.Context, rows []schema.KeyVerificationRequestV1) {
					table := "verifications.raw_key_verifications_v1"
					err := flush(ctx, conn, table, rows)
					if err != nil {
						config.Logger.Error("failed to flush batch",
							"table", table,
							"error", err.Error(),
						)
					}
				},
			},
		),

		keyVerificationsV2: batch.New[schema.KeyVerificationV2](
			batch.Config[schema.KeyVerificationV2]{
				Name:          "key_verifications_v2",
				Drop:          true,
				BatchSize:     50_000,
				BufferSize:    200_000,
				FlushInterval: 5 * time.Second,
				Consumers:     2,
				Flush: func(ctx context.Context, rows []schema.KeyVerificationV2) {
					table := "default.key_verifications_raw_v2"
					err := flush(ctx, conn, table, rows)
					if err != nil {
						config.Logger.Error("failed to flush batch",
							"table", table,
							"error", err.Error(),
						)
					}
				},
			},
		),

		ratelimits: batch.New[schema.RatelimitRequestV1](
			batch.Config[schema.RatelimitRequestV1]{
				Name:          "ratelimits",
				Drop:          true,
				BatchSize:     50_000,
				BufferSize:    200_000,
				FlushInterval: 5 * time.Second,
				Consumers:     2,
				Flush: func(ctx context.Context, rows []schema.RatelimitRequestV1) {
					table := "ratelimits.raw_ratelimits_v1"
					err := flush(ctx, conn, table, rows)
					if err != nil {
						config.Logger.Error("failed to flush batch",
							"table", table,
							"error", err.Error(),
						)
					}
				},
			}),
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

// Shutdown gracefully closes the ClickHouse client, ensuring that any
// pending batches are flushed before shutting down.
//
// This method should be called during application shutdown to prevent
// data loss. It will wait for all batch processors to complete their
// current work and close their channels.
//
// Example:
//
//	err := clickhouse.Shutdown(ctx)
//	if err != nil {
//	    logger.Error("failed to shutdown clickhouse client", err)
//	}
func (c *clickhouse) Shutdown(ctx context.Context) error {
	c.requests.Close()
	err := c.conn.Close()
	if err != nil {
		return fault.Wrap(err, fault.Internal("clickhouse couldn't shut down"))
	}
	return nil
}

// BufferRequest adds an API request event to the buffer for batch processing.
// The event will be flushed to ClickHouse automatically based on the configured
// batch size and flush interval.
//
// This method is non-blocking if the buffer has available capacity. If the buffer
// is full and the Drop option is enabled (which is the default), the event will
// be silently dropped.
//
// Example:
//
//	ch.BufferRequest(schema.ApiRequestV1{
//	    RequestID:      requestID,
//	    Time:           time.Now().UnixMilli(),
//	    WorkspaceID:    workspaceID,
//	    Host:           r.Host,
//	    Method:         r.Method,
//	    Path:           r.URL.Path,
//	    ResponseStatus: status,
//	})
func (c *clickhouse) BufferRequest(req schema.ApiRequestV1) {
	c.requests.Buffer(req)
}

// BufferApiRequest adds an API request event to the buffer for batch processing.
// The event will be flushed to ClickHouse automatically based on the configured
// batch size and flush interval.
//
// This method is non-blocking if the buffer has available capacity. If the buffer
// is full and the Drop option is enabled (which is the default), the event will
// be silently dropped.
//
// Example:
//
//	ch.BufferApiRequest(schema.ApiRequestV2{
//	    RequestID:      requestID,
//	    Time:           time.Now().UnixMilli(),
//	    WorkspaceID:    workspaceID,
//	    Host:           r.Host,
//	    Method:         r.Method,
//	    Path:           r.URL.Path,
//	    ResponseStatus: status,
//	})
func (c *clickhouse) BufferApiRequest(req schema.ApiRequestV2) {
	c.apiRequests.Buffer(req)
}

// BufferKeyVerification adds a key verification event to the buffer for batch processing.
// The event will be flushed to ClickHouse automatically based on the configured
// batch size and flush interval.
//
// This method is non-blocking if the buffer has available capacity. If the buffer
// is full and the Drop option is enabled (which is the default), the event will
// be silently dropped.
//
// Example:
//
//	ch.BufferKeyVerification(schema.KeyVerificationRequestV1{
//	    RequestID:  requestID,
//	    Time:       time.Now().UnixMilli(),
//	    WorkspaceID: workspaceID,
//	    KeyID:      keyID,
//	    Outcome:    "success",
//	})
func (c *clickhouse) BufferKeyVerification(req schema.KeyVerificationRequestV1) {
	c.keyVerifications.Buffer(req)
}

// BufferKeyVerificationV2 adds a key verification event to the buffer for batch processing.
// The event will be flushed to ClickHouse automatically based on the configured
// batch size and flush interval.
//
// This method is non-blocking if the buffer has available capacity. If the buffer
// is full and the Drop option is enabled (which is the default), the event will
// be silently dropped.
//
// Example:
//
//	ch.BufferKeyVerificationV2(schema.KeyVerificationV2{
//	    RequestID:  requestID,
//	    Time:       time.Now().UnixMilli(),
//	    WorkspaceID: workspaceID,
//	    KeyID:      keyID,
//	    Outcome:    "success",
//	})
func (c *clickhouse) BufferKeyVerificationV2(req schema.KeyVerificationV2) {
	c.keyVerificationsV2.Buffer(req)
}

// BufferRatelimit adds a ratelimit event to the buffer for batch processing.
// The event will be flushed to ClickHouse automatically based on the configured
// batch size and flush interval.
//
// This method is non-blocking if the buffer has available capacity. If the buffer
// is full and the Drop option is enabled (which is the default), the event will
// be silently dropped.
//
// Example:
//
//	ch.BufferRatelimit(schema.RatelimitRequestV1{
//	    RequestID:      requestID,
//	    Time:           time.Now().UnixMilli(),
//	    WorkspaceID:    workspaceID,
//	    NamespaceID:    namespaceID,
//	    Identifier:     identifier,
//	    Passed:         passed,
//	})
func (c *clickhouse) BufferRatelimit(req schema.RatelimitRequestV1) {
	c.ratelimits.Buffer(req)
}

func (c *clickhouse) Conn() ch.Conn {
	return c.conn
}

// QueryToMaps executes a query and scans all rows into a slice of maps.
// Each map represents a row with column names as keys and values as ch.Dynamic.
// Returns fault-wrapped errors with appropriate codes for resource limits,
// user query errors, and system errors.
func (c *clickhouse) QueryToMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, WrapClickHouseError(err)
	}
	defer rows.Close()

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
			return nil, fault.Wrap(err,
				fault.Public("Failed to read query results"),
			)
		}

		// Build map from column names to values
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
func (c *clickhouse) Exec(ctx context.Context, sql string, args ...any) error {
	return c.conn.Exec(ctx, sql, args...)
}

func (c *clickhouse) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close closes the underlying ClickHouse connection.
func (c *clickhouse) Close() error {
	return c.conn.Close()
}
