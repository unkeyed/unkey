package middleware

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
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

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorPageInfo struct {
	Status  int
	Title   string
	Message string
}

func categorizeErrorTypeIngress(urn codes.URN, statusCode int, hasError bool) string {
	if statusCode >= 200 && statusCode < 300 {
		return "none"
	}

	if hasError {

		//nolint:exhaustive
		switch urn {
		case codes.Ingress.Proxy.GatewayTimeout.URN():
			return "customer"

		case codes.Ingress.Internal.InternalServerError.URN(),
			codes.Ingress.Internal.ConfigLoadFailed.URN(),
			codes.Ingress.Internal.InstanceLoadFailed.URN(),
			codes.Ingress.Routing.ConfigNotFound.URN(),
			codes.Ingress.Routing.DeploymentSelectionFailed.URN(),
			codes.Ingress.Proxy.ServiceUnavailable.URN(),
			codes.Ingress.Routing.NoRunningInstances.URN():
			return "platform"

		case codes.User.BadRequest.ClientClosedRequest.URN(),
			codes.User.BadRequest.RequestTimeout.URN():
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

	return "unknown"
}

func WithObservability(logger logging.Logger, region string) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			startTime := time.Now()

			ingressActiveRequests.WithLabelValues(region).Inc()
			defer ingressActiveRequests.WithLabelValues(region).Dec()

			err := next(ctx, s)

			statusCode := s.StatusCode()
			var errorType string
			var urn codes.URN
			hasError := err != nil

			if hasError {
				var ok bool
				urn, ok = fault.GetCode(err)
				if !ok {
					urn = codes.Ingress.Internal.InternalServerError.URN()
				}

				code, parseErr := codes.ParseURN(urn)
				if parseErr != nil {
					logger.Error("failed to parse error code", "error", parseErr.Error())
					code = codes.Ingress.Internal.InternalServerError
				}

				pageInfo := getErrorPageInfoIngress(urn)
				statusCode = pageInfo.Status

				errorType = categorizeErrorTypeIngress(urn, statusCode, hasError)

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				title := pageInfo.Title

				if pageInfo.Status == http.StatusInternalServerError {
					logger.Error("ingress error",
						"error", err.Error(),
						"requestId", s.RequestID(),
						"publicMessage", userMessage,
						"status", pageInfo.Status,
						"path", s.Request().URL.Path,
						"host", s.Request().Host,
					)
				}

				acceptHeader := s.Request().Header.Get("Accept")
				preferJSON := strings.Contains(acceptHeader, "application/json") ||
					strings.Contains(acceptHeader, "application/*") ||
					(strings.Contains(acceptHeader, "*/*") && !strings.Contains(acceptHeader, "text/html"))

				var writeErr error
				if preferJSON {
					writeErr = s.JSON(pageInfo.Status, ErrorResponse{
						Error: ErrorDetail{
							Code:    string(code.URN()),
							Message: userMessage,
						},
					})
				} else {
					writeErr = s.HTML(pageInfo.Status, renderErrorHTMLIngress(title, userMessage, string(code.URN())))
				}

				if writeErr != nil {
					logger.Error("failed to write error response", "error", writeErr.Error())
				}
			} else {
				errorType = categorizeErrorTypeIngress("", statusCode, hasError)
			}

			duration := time.Since(startTime).Seconds()
			statusStr := strconv.Itoa(statusCode)

			logger.Info("ingress request",
				"status_code", statusStr,
				"error_type", errorType,
				"duration_seconds", duration,
				"region", region,
			)

			ingressRequestsTotal.WithLabelValues(statusStr, errorType, region).Inc()
			ingressRequestDuration.WithLabelValues(statusStr, errorType, region).Observe(duration)

			return nil
		}
	}
}

func getErrorPageInfoIngress(urn codes.URN) errorPageInfo {
	//nolint:exhaustive
	switch urn {
	case codes.User.BadRequest.ClientClosedRequest.URN():
		return errorPageInfo{
			Status:  499,
			Title:   "Client Closed Request",
			Message: "The client closed the connection before the request completed.",
		}
	case codes.Ingress.Routing.ConfigNotFound.URN():
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Title:   http.StatusText(http.StatusNotFound),
			Message: "No deployment found for this hostname. Please check your domain configuration or contact support.",
		}
	case codes.Ingress.Proxy.BadGateway.URN(),
		codes.Ingress.Proxy.ProxyForwardFailed.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Title:   http.StatusText(http.StatusBadGateway),
			Message: "Unable to connect. Please try again in a few moments.",
		}
	case codes.Ingress.Proxy.ServiceUnavailable.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Title:   http.StatusText(http.StatusServiceUnavailable),
			Message: "The service is temporarily unavailable. Please try again later.",
		}
	case codes.Ingress.Proxy.GatewayTimeout.URN():
		return errorPageInfo{
			Status:  http.StatusGatewayTimeout,
			Title:   http.StatusText(http.StatusGatewayTimeout),
			Message: "The request took too long to process. Please try again later.",
		}
	default:
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Title:   http.StatusText(http.StatusInternalServerError),
			Message: "",
		}
	}
}

func renderErrorHTMLIngress(title, message, errorCode string) []byte {
	escapedTitle := html.EscapeString(title)
	escapedMessage := html.EscapeString(message)
	escapedErrorCode := html.EscapeString(errorCode)

	return fmt.Appendf(nil, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <p>%s</p>
    <p class="error-code">Error: %s</p>
</body>
</html>`, escapedTitle, escapedTitle, escapedMessage, escapedErrorCode)
}
