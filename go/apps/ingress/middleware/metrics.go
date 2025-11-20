package middleware

import (
	"context"
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
	ingressRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingress_requests_total",
			Help: "Total number of requests processed by ingress",
		},
		[]string{"status_code", "error_type", "region"},
	)

	ingressRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ingress_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status_code", "error_type", "region"},
	)

	ingressActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ingress_active_requests",
			Help: "Number of requests currently being processed",
		},
		[]string{"region"},
	)
)

// categorizeErrorType determines if an error is a customer issue or platform issue
//
// When hasError is true: we categorize based on our error codes (platform vs customer infrastructure)
// When hasError is false: we use the captured status code to determine if customer's instance returned an error
func categorizeErrorType(urn codes.URN, statusCode int, hasError bool) string {
	// Success
	if statusCode >= 200 && statusCode < 300 {
		return "none"
	}

	// If we have an error from our code, check the error type
	if hasError {
		// Customer errors (their code/instance issues)
		switch urn {
		case codes.Gateway.Proxy.GatewayTimeout.URN(), // Instance timeout
			codes.Gateway.Proxy.ServiceUnavailable.URN(), // Instance unavailable
			codes.Gateway.Proxy.BadGateway.URN(),         // Instance returned invalid response
			codes.Ingress.Proxy.GatewayTimeout.URN(),     // Gateway/instance timeout
			codes.Ingress.Proxy.ServiceUnavailable.URN(), // Gateway/instance unavailable
			codes.Ingress.Routing.NoRunningInstances.URN(): // No instances running
			return "customer"

		// Platform errors (our infrastructure issues)
		case codes.Ingress.Internal.InternalServerError.URN(),
			codes.Ingress.Internal.ConfigLoadFailed.URN(),
			codes.Ingress.Internal.InstanceLoadFailed.URN(),
			codes.Ingress.Routing.ConfigNotFound.URN(),
			codes.Ingress.Routing.DeploymentSelectionFailed.URN():
			return "platform"

		// User errors (bad requests)
		case codes.User.BadRequest.ClientClosedRequest.URN(),
			codes.User.BadRequest.MissingRequiredHeader.URN(),
			codes.User.BadRequest.RequestTimeout.URN():
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

	return "unknown"
}

// WithMetrics returns middleware that records Prometheus metrics for requests
//
// This middleware tracks all request outcomes:
// - Errors from our code (platform/customer/user) via fault codes
// - Customer instance 4xx/5xx responses via status code capturing
func WithMetrics(logger logging.Logger, region string) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			startTime := time.Now()

			// Increment active requests
			ingressActiveRequests.WithLabelValues(region).Inc()
			defer ingressActiveRequests.WithLabelValues(region).Dec()

			// Process request
			err := next(ctx, s)

			// Get the status code (captured automatically by zen.Session)
			statusCode := s.StatusCode()
			errorType := "none"
			var urn codes.URN
			hasError := err != nil

			if err != nil {
				// Get the error URN
				var ok bool
				urn, ok = fault.GetCode(err)
				if !ok {
					urn = codes.Ingress.Internal.InternalServerError.URN()
				}

				// Get status code from error page info
				pageInfo := getErrorPageInfo(urn)
				statusCode = pageInfo.Status

				// Categorize error type
				errorType = categorizeErrorType(urn, statusCode, hasError)
			} else {
				// No error from our code, but check if customer's instance returned error status
				errorType = categorizeErrorType("", statusCode, hasError)
			}

			// Record metrics
			duration := time.Since(startTime).Seconds()
			statusStr := strconv.Itoa(statusCode)

			logger.Info("ingress request metrics",
				"status_code", statusStr,
				"error_type", errorType,
				"duration_seconds", duration,
				"region", region,
			)

			ingressRequestsTotal.WithLabelValues(statusStr, errorType, region).Inc()
			ingressRequestDuration.WithLabelValues(statusStr, errorType, region).Observe(duration)

			return err
		}
	}
}
