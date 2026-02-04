package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
	"go.opentelemetry.io/otel/attribute"
)

var (
	sentinelRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_requests_total",
			Help: "Total number of requests processed by sentinel",
		},
		[]string{"status_code", "error_type", "environment_id", "region"},
	)

	sentinelRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sentinel_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status_code", "error_type", "environment_id", "region"},
	)

	sentinelActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_active_requests",
			Help: "Number of requests currently being processed",
		},
		[]string{"environment_id", "region"},
	)
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorPageInfo struct {
	Status  int
	Message string
}

func getErrorPageInfo(urn codes.URN) errorPageInfo {
	//nolint:exhaustive
	switch urn {
	// Sentinel Routing Errors
	case codes.Sentinel.Routing.DeploymentNotFound.URN():
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Message: "The requested deployment could not be found.",
		}
	case codes.Sentinel.Routing.NoRunningInstances.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Message: "No running instances are available to handle this request.",
		}
	case codes.Sentinel.Routing.InstanceSelectionFailed.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "Failed to select an instance to handle your request.",
		}

	// Sentinel Proxy Errors
	case codes.Sentinel.Proxy.SentinelTimeout.URN():
		return errorPageInfo{
			Status:  http.StatusGatewayTimeout,
			Message: "The request took too long to process. Please try again later.",
		}
	case codes.Sentinel.Proxy.ServiceUnavailable.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Message: "The service is temporarily unavailable. Please try again later.",
		}
	case codes.Sentinel.Proxy.BadGateway.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Message: "Unable to connect to an instance. Please try again in a few moments.",
		}
	case codes.Sentinel.Proxy.ProxyForwardFailed.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Message: "Failed to forward the request. Please try again.",
		}

	// Sentinel Internal Errors
	case codes.Sentinel.Internal.InternalServerError.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "An unexpected error occurred. Please try again later.",
		}
	case codes.Sentinel.Internal.InvalidConfiguration.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "The sentinel is misconfigured. Please contact support.",
		}

	// User/Client Errors
	case codes.User.BadRequest.ClientClosedRequest.URN():
		return errorPageInfo{
			Status:  499, // Non-standard but commonly used for client closed request
			Message: "The client closed the connection before the request completed.",
		}
	case codes.User.BadRequest.MissingRequiredHeader.URN():
		return errorPageInfo{
			Status:  http.StatusBadRequest,
			Message: "A required header is missing from the request.",
		}

	// Default
	default:
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "An unexpected error occurred. Please try again later.",
		}
	}
}

func categorizeErrorType(urn codes.URN, statusCode int, hasError bool) string {
	if statusCode >= 200 && statusCode < 300 {
		return "none"
	}

	if hasError {
		//nolint:exhaustive
		switch urn {
		case codes.Sentinel.Proxy.SentinelTimeout.URN(),
			codes.Sentinel.Proxy.BadGateway.URN(),
			codes.Sentinel.Proxy.ProxyForwardFailed.URN():
			return "customer"

		case codes.Sentinel.Internal.InternalServerError.URN(),
			codes.Sentinel.Internal.InvalidConfiguration.URN(),
			codes.Sentinel.Routing.DeploymentNotFound.URN(),
			codes.Sentinel.Routing.InstanceSelectionFailed.URN(),
			codes.Sentinel.Proxy.ServiceUnavailable.URN(),
			codes.Sentinel.Routing.NoRunningInstances.URN():
			return "platform"

		case codes.User.BadRequest.ClientClosedRequest.URN(),
			codes.User.BadRequest.MissingRequiredHeader.URN():
			return "user"
		}

		if statusCode >= 500 {
			return "platform"
		}

		if statusCode >= 400 {
			return "user"
		}
	} else if statusCode >= 400 {
		return "customer"
	}

	return "none"
}

func WithObservability(logger logging.Logger, environmentID, region string) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			startTime := time.Now()

			// Start trace span for the request
			ctx, span := tracing.Start(ctx, "sentinel.proxy")
			span.SetAttributes(
				attribute.String("request_id", s.RequestID()),
				attribute.String("host", s.Request().Host),
				attribute.String("method", s.Request().Method),
				attribute.String("path", s.Request().URL.Path),
				attribute.String("environment_id", environmentID),
				attribute.String("region", region),
			)
			defer span.End()

			sentinelActiveRequests.WithLabelValues(environmentID, region).Inc()
			defer sentinelActiveRequests.WithLabelValues(environmentID, region).Dec()

			err := next(ctx, s)

			statusCode := s.StatusCode()
			var errorType string
			var urn codes.URN
			hasError := err != nil

			if hasError {
				tracing.RecordError(span, err)

				var ok bool
				urn, ok = fault.GetCode(err)
				if !ok {
					urn = codes.Sentinel.Internal.InternalServerError.URN()
				}

				code, parseErr := codes.ParseURN(urn)
				if parseErr != nil {
					logger.Error("failed to parse error code", "error", parseErr.Error())
					code = codes.Sentinel.Internal.InternalServerError
				}

				pageInfo := getErrorPageInfo(urn)
				statusCode = pageInfo.Status

				errorType = categorizeErrorType(urn, statusCode, hasError)

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				if pageInfo.Status == http.StatusInternalServerError {
					logger.Error("sentinel error",
						"error", err.Error(),
						"requestId", s.RequestID(),
						"publicMessage", userMessage,
						"status", pageInfo.Status,
						"path", s.Request().URL.Path,
						"host", s.Request().Host,
					)
				}

				s.ResponseWriter().Header().Set("X-Unkey-Error-Source", "sentinel")

				writeErr := s.JSON(pageInfo.Status, ErrorResponse{
					Error: ErrorDetail{
						Code:    string(code.URN()),
						Message: userMessage,
					},
				})
				if writeErr != nil {
					logger.Error("failed to write error response", "error", writeErr.Error())
				}
			} else {
				errorType = categorizeErrorType("", statusCode, hasError)
			}

			duration := time.Since(startTime).Seconds()
			statusStr := strconv.Itoa(statusCode)

			// Add final status to span
			span.SetAttributes(
				attribute.Int("status_code", statusCode),
				attribute.String("error_type", errorType),
			)

			logger.Info("sentinel request",
				"status_code", statusStr,
				"error_type", errorType,
				"duration_seconds", duration,
				"environment_id", environmentID,
				"region", region,
			)

			sentinelRequestsTotal.WithLabelValues(statusStr, errorType, environmentID, region).Inc()
			sentinelRequestDuration.WithLabelValues(statusStr, errorType, environmentID, region).Observe(duration)

			return nil
		}
	}
}

// WideObservabilityConfig holds configuration for wide-enabled observability.
type WideObservabilityConfig struct {
	Logger         logging.Logger
	EnvironmentID  string
	Region         string
	ServiceVersion string
	Sampler        wide.Sampler
}

// WithWideObservability returns an observability middleware that uses wide for wide event logging.
// This replaces the standard logging with a single comprehensive log per request.
func WithWideObservability(config WideObservabilityConfig) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			// Create EventContext with logger and sampler
			ctx, ev := wide.WithEventContext(ctx, wide.EventConfig{
				Name:    s.Request().Method + " " + s.Request().URL.Path,
				Logger:  config.Logger,
				Sampler: config.Sampler,
			})

			// Capture initial request metadata
			ev.SetMany(map[string]any{
				wide.FieldRequestID:      s.RequestID(),
				wide.FieldMethod:         s.Request().Method,
				wide.FieldPath:           s.Request().URL.Path,
				wide.FieldHost:           s.Request().Host,
				wide.FieldUserAgent:      s.UserAgent(),
				wide.FieldIPAddress:      s.Location(),
				wide.FieldServiceName:    "sentinel",
				wide.FieldServiceVersion: config.ServiceVersion,
				wide.FieldEnvironmentID:  config.EnvironmentID,
				wide.FieldRegion:         config.Region,
			})

			// Start trace span for the request
			ctx, span := tracing.Start(ctx, "sentinel.proxy")
			span.SetAttributes(
				attribute.String("request_id", s.RequestID()),
				attribute.String("host", s.Request().Host),
				attribute.String("method", s.Request().Method),
				attribute.String("path", s.Request().URL.Path),
				attribute.String("environment_id", config.EnvironmentID),
				attribute.String("region", config.Region),
			)
			defer span.End()

			sentinelActiveRequests.WithLabelValues(config.EnvironmentID, config.Region).Inc()
			defer sentinelActiveRequests.WithLabelValues(config.EnvironmentID, config.Region).Dec()

			err := next(ctx, s)

			statusCode := s.StatusCode()
			var errorType string
			var urn codes.URN
			hasError := err != nil

			if hasError {
				tracing.RecordError(span, err)
				ev.MarkError()

				var ok bool
				urn, ok = fault.GetCode(err)
				if !ok {
					urn = codes.Sentinel.Internal.InternalServerError.URN()
				}

				code, parseErr := codes.ParseURN(urn)
				if parseErr != nil {
					config.Logger.Error("failed to parse error code", "error", parseErr.Error())
					code = codes.Sentinel.Internal.InternalServerError
				}

				pageInfo := getErrorPageInfo(urn)
				statusCode = pageInfo.Status

				errorType = categorizeErrorType(urn, statusCode, hasError)

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				// Add error details to wide
				ev.Set(wide.FieldError, userMessage)
				ev.Set(wide.FieldErrorCode, string(code.URN()))
				ev.Set(wide.FieldErrorInternal, err.Error())

				s.ResponseWriter().Header().Set("X-Unkey-Error-Source", "sentinel")

				writeErr := s.JSON(pageInfo.Status, ErrorResponse{
					Error: ErrorDetail{
						Code:    string(code.URN()),
						Message: userMessage,
					},
				})
				if writeErr != nil {
					config.Logger.Error("failed to write error response", "error", writeErr.Error())
				}
			} else {
				errorType = categorizeErrorType("", statusCode, hasError)
			}

			duration := ev.Duration()
			durationSeconds := duration.Seconds()
			statusStr := strconv.Itoa(statusCode)

			// Add final metrics to wide
			ev.Set(wide.FieldStatusCode, statusCode)
			ev.Set(wide.FieldDurationMs, duration.Milliseconds())
			ev.Set(wide.FieldErrorType, errorType)

			// Add final status to span
			span.SetAttributes(
				attribute.Int("status_code", statusCode),
				attribute.String("error_type", errorType),
			)

			// Emit the event (sampling is handled internally)
			ev.Emit()

			sentinelRequestsTotal.WithLabelValues(statusStr, errorType, config.EnvironmentID, config.Region).Inc()
			sentinelRequestDuration.WithLabelValues(statusStr, errorType, config.EnvironmentID, config.Region).Observe(durationSeconds)

			return nil
		}
	}
}
