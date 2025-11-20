package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

var (
	gatewayRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests processed by gateway",
		},
		[]string{"status_code", "error_type", "environment_id", "region"},
	)

	gatewayRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status_code", "error_type", "environment_id", "region"},
	)

	gatewayActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_active_requests",
			Help: "Number of requests currently being processed",
		},
		[]string{"environment_id", "region"},
	)
)

// ErrorResponse is the standard JSON error response format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// errorPageInfo holds the HTTP status and message for an error.
type errorPageInfo struct {
	Status  int
	Message string
}

// getErrorPageInfo returns the HTTP status and user-friendly message for an error URN.
func getErrorPageInfo(urn codes.URN) errorPageInfo {
	switch urn {
	// Gateway Routing Errors
	case codes.Gateway.Routing.DeploymentNotFound.URN():
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Message: "The requested deployment could not be found.",
		}
	case codes.Gateway.Routing.NoRunningInstances.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Message: "No running instances are available to handle this request.",
		}
	case codes.Gateway.Routing.InstanceSelectionFailed.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "Failed to select an instance to handle your request.",
		}

	// Gateway Proxy Errors
	case codes.Gateway.Proxy.GatewayTimeout.URN():
		return errorPageInfo{
			Status:  http.StatusGatewayTimeout,
			Message: "The request took too long to process. Please try again later.",
		}
	case codes.Gateway.Proxy.ServiceUnavailable.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Message: "The service is temporarily unavailable. Please try again later.",
		}
	case codes.Gateway.Proxy.BadGateway.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Message: "Unable to connect to the backend service. Please try again in a few moments.",
		}
	case codes.Gateway.Proxy.ProxyForwardFailed.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Message: "Failed to forward the request. Please try again.",
		}

	// Gateway Internal Errors
	case codes.Gateway.Internal.InternalServerError.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "An unexpected error occurred. Please try again later.",
		}
	case codes.Gateway.Internal.InvalidConfiguration.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "The gateway is misconfigured. Please contact support.",
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

// categorizeErrorType determines if an error is a customer issue or platform issue
//
// When hasError is true: we categorize based on our error codes (platform vs customer infrastructure)
// When hasError is false: we use the captured status code to determine if customer's instance returned an error
func categorizeErrorType(urn codes.URN, statusCode int, hasError bool) string {
	// Success
	if statusCode >= 200 && statusCode < 300 {
		return "none"
	}

	// If we have an error from our code, categorize it
	if hasError {
		// Customer errors (their code/instance issues)
		switch urn {
		case codes.Gateway.Proxy.GatewayTimeout.URN(),    // Instance timeout
			codes.Gateway.Proxy.BadGateway.URN(),         // Instance returned invalid response
			codes.Gateway.Proxy.ProxyForwardFailed.URN(): // Failed to forward to instance
			return "customer"

		// Platform errors (our infrastructure issues)
		case codes.Gateway.Internal.InternalServerError.URN(),
			codes.Gateway.Internal.InvalidConfiguration.URN(),
			codes.Gateway.Routing.DeploymentNotFound.URN(),
			codes.Gateway.Routing.InstanceSelectionFailed.URN(),
			codes.Gateway.Proxy.ServiceUnavailable.URN(),    // Connection refused - we should health check
			codes.Gateway.Routing.NoRunningInstances.URN():  // No instances - orchestration issue
			return "platform"

		// User errors (bad requests)
		case codes.User.BadRequest.ClientClosedRequest.URN(),
			codes.User.BadRequest.MissingRequiredHeader.URN():
			return "user"
		}

		// Default for errors: if 5xx, assume platform; if 4xx, assume user
		if statusCode >= 500 {
			return "platform"
		}

		if statusCode >= 400 {
			return "user"
		}
	} else {
		// No error from our code, but non-2xx status = customer's code returned error
		// This happens when we successfully proxy, but instance returns 4xx/5xx
		if statusCode >= 500 {
			return "customer" // Customer's instance returned 5xx
		}
		if statusCode >= 400 {
			return "customer" // Customer's instance returned 4xx
		}
	}

	return "none"
}

// WithObservability combines error handling and metrics into a single middleware.
// This handles errors, writes appropriate responses, and records metrics for monitoring.
func WithObservability(logger logging.Logger, environmentID, region string) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			startTime := time.Now()

			// Increment active requests
			gatewayActiveRequests.WithLabelValues(environmentID, region).Inc()
			defer gatewayActiveRequests.WithLabelValues(environmentID, region).Dec()

			// Process request
			err := next(ctx, s)

			// Get the status code (captured automatically by zen.Session)
			statusCode := s.StatusCode()
			errorType := "none"
			var urn codes.URN
			hasError := err != nil

			// Handle errors
			if hasError {
				// Get the error URN
				var ok bool
				urn, ok = fault.GetCode(err)
				if !ok {
					urn = codes.Gateway.Internal.InternalServerError.URN()
				}

				code, parseErr := codes.ParseURN(urn)
				if parseErr != nil {
					logger.Error("failed to parse error code", "error", parseErr.Error())
					code = codes.Gateway.Internal.InternalServerError
				}

				// Get error page info (status and message) based on URN
				pageInfo := getErrorPageInfo(urn)
				statusCode = pageInfo.Status

				// Categorize error type for metrics
				errorType = categorizeErrorType(urn, statusCode, hasError)

				// Use user-facing message from error if no specific message defined
				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				// Log internal server errors
				if pageInfo.Status == http.StatusInternalServerError {
					logger.Error("gateway error",
						"error", err.Error(),
						"requestId", s.RequestID(),
						"publicMessage", userMessage,
						"status", pageInfo.Status,
						"path", s.Request().URL.Path,
						"host", s.Request().Host,
					)
				}

				// Write error response (gateway is not user-facing, only ingress calls it)
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
				// No error from our code, but check if customer's instance returned error status
				errorType = categorizeErrorType("", statusCode, hasError)
			}

			// Record metrics
			duration := time.Since(startTime).Seconds()
			statusStr := strconv.Itoa(statusCode)

			logger.Info("gateway request",
				"status_code", statusStr,
				"error_type", errorType,
				"duration_seconds", duration,
				"environment_id", environmentID,
				"region", region,
			)

			gatewayRequestsTotal.WithLabelValues(statusStr, errorType, environmentID, region).Inc()
			gatewayRequestDuration.WithLabelValues(statusStr, errorType, environmentID, region).Observe(duration)

			// Return nil - error has been handled
			return nil
		}
	}
}
