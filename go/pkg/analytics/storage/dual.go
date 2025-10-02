package storage

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/analytics"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// DualWriter implements the Writer interface by writing to two different writers.
// This allows writing to multiple backends simultaneously (e.g., ClickHouse + Data Lake).
type DualWriter struct {
	primary   analytics.Writer
	secondary analytics.Writer
	config    DualConfig
	logger    logging.Logger
}

// DualConfig contains configuration for dual writer behavior.
type DualConfig struct {
	// FailOnPrimaryError determines if writes should fail when primary writer fails
	FailOnPrimaryError bool

	// FailOnSecondaryError determines if writes should fail when secondary writer fails
	FailOnSecondaryError bool
}

// NewDualWriter creates a new dual writer that writes to both primary and secondary writers.
func NewDualWriter(primary, secondary analytics.Writer, config DualConfig, logger logging.Logger) analytics.Writer {
	return &DualWriter{
		primary:   primary,
		secondary: secondary,
		config:    config,
		logger:    logger,
	}
}

// KeyVerification writes the key verification event to both writers.
func (w *DualWriter) KeyVerification(ctx context.Context, data schema.KeyVerificationV2) error {
	var primaryErr, secondaryErr error

	// Write to primary
	primaryErr = w.primary.KeyVerification(ctx, data)
	if primaryErr != nil {
		w.logger.Error("primary writer failed for key verification",
			"error", primaryErr.Error(),
			"request_id", data.RequestID,
		)
	}

	// Write to secondary
	secondaryErr = w.secondary.KeyVerification(ctx, data)
	if secondaryErr != nil {
		w.logger.Error("secondary writer failed for key verification",
			"error", secondaryErr.Error(),
			"request_id", data.RequestID,
		)
	}

	// Return error based on configuration
	if primaryErr != nil && w.config.FailOnPrimaryError {
		return primaryErr
	}
	if secondaryErr != nil && w.config.FailOnSecondaryError {
		return secondaryErr
	}

	return nil
}

// Ratelimit writes the ratelimit event to both writers.
func (w *DualWriter) Ratelimit(ctx context.Context, data schema.RatelimitV2) error {
	var primaryErr, secondaryErr error

	// Write to primary
	primaryErr = w.primary.Ratelimit(ctx, data)
	if primaryErr != nil {
		w.logger.Error("primary writer failed for ratelimit",
			"error", primaryErr.Error(),
			"request_id", data.RequestID,
		)
	}

	// Write to secondary
	secondaryErr = w.secondary.Ratelimit(ctx, data)
	if secondaryErr != nil {
		w.logger.Error("secondary writer failed for ratelimit",
			"error", secondaryErr.Error(),
			"request_id", data.RequestID,
		)
	}

	// Return error based on configuration
	if primaryErr != nil && w.config.FailOnPrimaryError {
		return primaryErr
	}
	if secondaryErr != nil && w.config.FailOnSecondaryError {
		return secondaryErr
	}

	return nil
}

// ApiRequest writes the API request event to both writers.
func (w *DualWriter) ApiRequest(ctx context.Context, data schema.ApiRequestV2) error {
	var primaryErr, secondaryErr error

	// Write to primary
	primaryErr = w.primary.ApiRequest(ctx, data)
	if primaryErr != nil {
		w.logger.Error("primary writer failed for api request",
			"error", primaryErr.Error(),
			"request_id", data.RequestID,
		)
	}

	// Write to secondary
	secondaryErr = w.secondary.ApiRequest(ctx, data)
	if secondaryErr != nil {
		w.logger.Error("secondary writer failed for api request",
			"error", secondaryErr.Error(),
			"request_id", data.RequestID,
		)
	}

	// Return error based on configuration
	if primaryErr != nil && w.config.FailOnPrimaryError {
		return primaryErr
	}
	if secondaryErr != nil && w.config.FailOnSecondaryError {
		return secondaryErr
	}

	return nil
}

// Close gracefully closes both writers.
func (w *DualWriter) Close(ctx context.Context) error {
	w.logger.Info("closing dual writer")

	var primaryErr, secondaryErr error

	// Close primary
	primaryErr = w.primary.Close(ctx)
	if primaryErr != nil {
		w.logger.Error("failed to close primary writer", "error", primaryErr.Error())
	}

	// Close secondary
	secondaryErr = w.secondary.Close(ctx)
	if secondaryErr != nil {
		w.logger.Error("failed to close secondary writer", "error", secondaryErr.Error())
	}

	// Return first error encountered
	if primaryErr != nil {
		return primaryErr
	}
	return secondaryErr
}
