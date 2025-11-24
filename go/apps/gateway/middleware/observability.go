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
			Message: "Unable to connect to a instance. Please try again in a few moments.",
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

func categorizeErrorType(urn codes.URN, statusCode int, hasError bool) string {
	if statusCode >= 200 && statusCode < 300 {
		return "none"
	}

	if hasError {
		switch urn {
		case codes.Gateway.Proxy.GatewayTimeout.URN(),
			codes.Gateway.Proxy.BadGateway.URN(),
			codes.Gateway.Proxy.ProxyForwardFailed.URN():
			return "customer"

		case codes.Gateway.Internal.InternalServerError.URN(),
			codes.Gateway.Internal.InvalidConfiguration.URN(),
			codes.Gateway.Routing.DeploymentNotFound.URN(),
			codes.Gateway.Routing.InstanceSelectionFailed.URN(),
			codes.Gateway.Proxy.ServiceUnavailable.URN(),
			codes.Gateway.Routing.NoRunningInstances.URN():
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

			gatewayActiveRequests.WithLabelValues(environmentID, region).Inc()
			defer gatewayActiveRequests.WithLabelValues(environmentID, region).Dec()

			err := next(ctx, s)

			statusCode := s.StatusCode()
			errorType := "none"
			var urn codes.URN
			hasError := err != nil

			if hasError {
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

				pageInfo := getErrorPageInfo(urn)
				statusCode = pageInfo.Status

				errorType = categorizeErrorType(urn, statusCode, hasError)

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

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

				s.ResponseWriter().Header().Set("X-Unkey-Error-Source", "gateway")

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

			logger.Info("gateway request",
				"status_code", statusStr,
				"error_type", errorType,
				"duration_seconds", duration,
				"environment_id", environmentID,
				"region", region,
			)

			gatewayRequestsTotal.WithLabelValues(statusStr, errorType, environmentID, region).Inc()
			gatewayRequestDuration.WithLabelValues(statusStr, errorType, environmentID, region).Observe(duration)

			return nil
		}
	}
}
