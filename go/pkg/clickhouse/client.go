package clickhouse

import (
	"context"
	"fmt"
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
type Clickhouse struct {
	conn   ch.Conn
	logger logging.Logger

	// Batched processors for different event types
	requests         *batch.BatchProcessor[schema.ApiRequestV1]
	keyVerifications *batch.BatchProcessor[schema.KeyVerificationRequestV1]
}

var _ Bufferer = (*Clickhouse)(nil)
var _ Querier = (*Clickhouse)(nil)

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
func New(config Config) (*Clickhouse, error) {
	opts, err := ch.ParseDSN(config.URL)
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("parsing clickhouse DSN failed", ""))
	}

	// opts.TLS = &tls.Config{}
	opts.Debug = true
	opts.Debugf = func(format string, v ...any) {
		config.Logger.Debug(fmt.Sprintf(format, v...))
	}
	//	if opts.TLS == nil {
	//
	//		opts.TLS = new(tls.Config)
	//	}

	config.Logger.Info("connecting to clickhouse")
	conn, err := ch.Open(opts)
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("opening clickhouse failed", ""))

	}

	err = retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
	).
		Do(func() error {
			return conn.Ping(context.Background())
		})
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("pinging clickhouse failed", ""))
	}
	c := &Clickhouse{
		conn:   conn,
		logger: config.Logger,

		requests: batch.New[schema.ApiRequestV1](batch.Config[schema.ApiRequestV1]{
			Name:          "api reqeusts",
			Drop:          true,
			BatchSize:     1000,
			BufferSize:    100000,
			FlushInterval: time.Second,
			Consumers:     4,
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
		keyVerifications: batch.New[schema.KeyVerificationRequestV1](
			batch.Config[schema.KeyVerificationRequestV1]{
				Name:          "key verifications",
				Drop:          true,
				BatchSize:     1000,
				BufferSize:    100000,
				FlushInterval: time.Second,
				Consumers:     4,
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
			}),
	}

	// err = c.conn.Ping(context.Background())
	// if err != nil {
	// 	return nil, fault.Wrap(err, fault.With("pinging clickhouse failed"))
	// }
	return c, nil
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
func (c *Clickhouse) Shutdown(ctx context.Context) error {
	c.requests.Close()
	err := c.conn.Close()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("clickhouse couldn't shut down", ""))
	}
	return nil
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
//	ch.BufferApiRequest(schema.ApiRequestV1{
//	    RequestID:      requestID,
//	    Time:           time.Now().UnixMilli(),
//	    WorkspaceID:    workspaceID,
//	    Host:           r.Host,
//	    Method:         r.Method,
//	    Path:           r.URL.Path,
//	    ResponseStatus: status,
//	})
func (c *Clickhouse) BufferApiRequest(req schema.ApiRequestV1) {
	c.requests.Buffer(req)
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
func (c *Clickhouse) BufferKeyVerification(req schema.KeyVerificationRequestV1) {
	c.keyVerifications.Buffer(req)
}
