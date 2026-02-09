// Package clickhouse provides a client for interacting with ClickHouse databases,
// optimized for high-volume event data storage and analytics.
//
// It implements efficient batch processing for different event types, with
// support for buffering, automatic retries, and graceful shutdown. The package
// is designed to handle high-throughput logging scenarios where individual
// event latency is less critical than overall throughput and reliability.
//
// Key features:
//   - Batched writes to minimize network overhead
//   - Buffer-based queueing to handle traffic spikes
//   - Automatic connection management
//   - Graceful shutdown with final flush capability
//   - Support for multiple event types with dedicated buffers
//
// Example usage:
//
//	// Create a ClickHouse client
//	ch, err := clickhouse.New(clickhouse.Config{
//	    URL:    "clickhouse://user:pass@clickhouse.example.com:9000/db?secure=true",
//	})
//	if err != nil {
//	    return fmt.Errorf("failed to create clickhouse client: %w", err)
//	}
//
//	// Buffer events for batch processing
//	ch.BufferRequest(schema.ApiRequestV1{
//	    RequestID:      "req_123",
//	    Time:           time.Now().UnixMilli(),
//	    WorkspaceID:    "ws_abc",
//	    Host:           "api.example.com",
//	    Method:         "POST",
//	    Path:           "/v1/keys/verify",
//	    ResponseStatus: 200,
//	})
//
//	// Events are automatically flushed based on batch size and interval
//	// When shutting down:
//	err = ch.Shutdown(ctx)
package clickhouse
