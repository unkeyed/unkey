package storage

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/analytics"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// NoopWriter implements the Writer interface but performs no operations.
// This is useful for testing, development, or when analytics are disabled.
type NoopWriter struct{}

// NewNoopWriter creates a new no-op writer that discards all events.
func NewNoopWriter() analytics.Writer {
	return &NoopWriter{}
}

// KeyVerification discards the key verification event.
func (w *NoopWriter) KeyVerification(ctx context.Context, data schema.KeyVerificationV2) error {
	// No-op: silently discard the event
	return nil
}

// Ratelimit discards the ratelimit event.
func (w *NoopWriter) Ratelimit(ctx context.Context, data schema.RatelimitV2) error {
	// No-op: silently discard the event
	return nil
}

// ApiRequest discards the API request event.
func (w *NoopWriter) ApiRequest(ctx context.Context, data schema.ApiRequestV2) error {
	// No-op: silently discard the event
	return nil
}

// Close performs no operation for cleanup.
func (w *NoopWriter) Close(ctx context.Context) error {
	// No-op: nothing to clean up
	return nil
}
