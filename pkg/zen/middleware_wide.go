package zen

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// WideConfig holds configuration for the wide middleware.
type WideConfig struct {
	// Logger is the logger to emit wide events to.
	Logger logging.Logger

	// Sampler controls which events get logged.
	// If nil, defaults to the standard tail sampler (errors + slow + 1% success).
	Sampler wide.Sampler

	// ServiceName is the name of the service (e.g., "api", "frontline", "sentinel").
	ServiceName string

	// ServiceVersion is the version/image of the service.
	ServiceVersion string

	// Region is the geographic region where the service is running.
	Region string
}

// WithWide returns middleware that implements "one log per request" wide event logging.
//
// This middleware:
//  1. Creates an EventContext at the start of each request
//  2. Populates initial request metadata (request_id, method, path, etc.)
//  3. Calls the next handler
//  4. Captures response metadata (status_code, duration_ms, etc.)
//  5. Emits a single wide event with all accumulated context (if sampled)
//
// Handlers and services can add additional context during processing:
//
//	wide.Set(ctx, "key_id", keyID)
//	wide.Set(ctx, "cache_hit", true)
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{
//	        zen.WithWide(zen.WideConfig{
//	            Logger:      logger,
//	            ServiceName: "api",
//	            Region:      "us-east-1",
//	        }),
//	    },
//	    route,
//	)
func WithWide(config WideConfig) Middleware {
	// Default to tail sampler if none provided
	sampler := config.Sampler
	if sampler == nil {
		sampler = wide.NewTailSampler(wide.DefaultTailSamplerConfig())
	}

	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			// Create EventContext with logger and sampler
			ctx, ev := wide.WithEventContext(ctx, wide.EventConfig{
				Logger:  config.Logger,
				Sampler: sampler,
			})

			// Capture initial request metadata
			ev.SetMany(map[string]any{
				wide.FieldRequestID:      s.RequestID(),
				wide.FieldMethod:         s.r.Method,
				wide.FieldPath:           s.r.URL.Path,
				wide.FieldHost:           s.r.Host,
				wide.FieldUserAgent:      s.UserAgent(),
				wide.FieldIPAddress:      s.Location(),
				wide.FieldServiceName:    config.ServiceName,
				wide.FieldServiceVersion: config.ServiceVersion,
				wide.FieldRegion:         config.Region,
			})

			// Add content length if present
			if s.r.ContentLength > 0 {
				ev.Set(wide.FieldContentLength, s.r.ContentLength)
			}

			// Execute the handler chain
			nextErr := next(ctx, s)

			// Capture response metadata
			statusCode := s.StatusCode()

			ev.Set(wide.FieldStatusCode, statusCode)
			ev.Set(wide.FieldDurationMs, ev.DurationMs())

			// Capture workspace ID if set
			if s.WorkspaceID != "" {
				ev.Set(wide.FieldWorkspaceID, s.WorkspaceID)
			}

			// Capture internal error if present
			if internalErr := s.InternalError(); internalErr != "" {
				ev.Set(wide.FieldErrorInternal, internalErr)
				ev.MarkError()
			}

			// Mark error if handler returned an error
			if nextErr != nil {
				ev.Set(wide.FieldError, nextErr.Error())
				ev.MarkError()
			}

			// Determine error type based on status code
			errorType := categorizeErrorType(statusCode, ev.HasError())
			ev.Set(wide.FieldErrorType, errorType)

			// Emit the event (sampling is handled internally)
			ev.Emit()

			return nextErr
		}
	}
}

// categorizeErrorType determines the error type based on status code and error presence.
func categorizeErrorType(statusCode int, hasError bool) string {
	if statusCode >= 200 && statusCode < 300 {
		return wide.ErrorTypeNone
	}

	if statusCode >= 500 {
		return wide.ErrorTypePlatform
	}

	if statusCode >= 400 {
		if hasError {
			return wide.ErrorTypeUser
		}
		// 4xx without internal error usually means customer's upstream error (for proxy services)
		return wide.ErrorTypeCustomer
	}

	return wide.ErrorTypeUnknown
}

// NewWideConfig creates an WideConfig with common settings using the default tail sampler.
func NewWideConfig(
	logger logging.Logger,
	serviceName string,
	serviceVersion string,
	region string,
) WideConfig {
	return WideConfig{
		Logger:         logger,
		Sampler:        wide.NewTailSampler(wide.DefaultTailSamplerConfig()),
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Region:         region,
	}
}

// NewWideConfigWithSampler creates an WideConfig with a custom sampler.
func NewWideConfigWithSampler(
	logger logging.Logger,
	sampler wide.Sampler,
	serviceName string,
	serviceVersion string,
	region string,
) WideConfig {
	return WideConfig{
		Logger:         logger,
		Sampler:        sampler,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Region:         region,
	}
}

// NewTailSamplerFromConfig creates a tail sampler from sample rate and threshold settings.
func NewTailSamplerFromConfig(successSampleRate float64, slowThresholdMs int) wide.Sampler {
	slowThreshold := time.Duration(slowThresholdMs) * time.Millisecond
	if slowThresholdMs <= 0 {
		slowThreshold = 500 * time.Millisecond
	}
	if successSampleRate <= 0 {
		successSampleRate = 0.01
	}

	return wide.NewTailSampler(wide.TailSamplerConfig{
		SuccessSampleRate: successSampleRate,
		SlowThreshold:     slowThreshold,
	})
}
