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
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
	"go.opentelemetry.io/otel/attribute"
)

var (
	frontlineRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "frontline_requests_total",
			Help: "Total number of requests processed by frontline",
		},
		[]string{"status_code", "error_type", "region"},
	)

	frontlineRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "frontline_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status_code", "error_type", "region"},
	)

	frontlineActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "frontline_active_requests",
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

func categorizeErrorTypeFrontline(urn codes.URN, statusCode int, hasError bool) string {
	if statusCode >= 200 && statusCode < 300 {
		return "none"
	}

	if hasError {

		//nolint:exhaustive
		switch urn {
		case codes.Frontline.Proxy.GatewayTimeout.URN():
			return "customer"

		case codes.Frontline.Internal.InternalServerError.URN(),
			codes.Frontline.Internal.ConfigLoadFailed.URN(),
			codes.Frontline.Internal.InstanceLoadFailed.URN(),
			codes.Frontline.Routing.ConfigNotFound.URN(),
			codes.Frontline.Routing.DeploymentSelectionFailed.URN(),
			codes.Frontline.Proxy.ServiceUnavailable.URN(),
			codes.Frontline.Routing.NoRunningInstances.URN():
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

			// Start trace span for the request
			ctx, span := tracing.Start(ctx, "frontline.proxy")
			span.SetAttributes(
				attribute.String("request_id", s.RequestID()),
				attribute.String("host", s.Request().Host),
				attribute.String("method", s.Request().Method),
				attribute.String("path", s.Request().URL.Path),
				attribute.String("region", region),
			)
			defer span.End()

			frontlineActiveRequests.WithLabelValues(region).Inc()
			defer frontlineActiveRequests.WithLabelValues(region).Dec()

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
					urn = codes.Frontline.Internal.InternalServerError.URN()
				}

				code, parseErr := codes.ParseURN(urn)
				if parseErr != nil {
					logger.Error("failed to parse error code", "error", parseErr.Error())
					code = codes.Frontline.Internal.InternalServerError
				}

				pageInfo := getErrorPageInfoFrontline(urn)
				statusCode = pageInfo.Status

				errorType = categorizeErrorTypeFrontline(urn, statusCode, hasError)

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				title := pageInfo.Title

				if pageInfo.Status == http.StatusInternalServerError {
					logger.Error("frontline error",
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
					writeErr = s.HTML(pageInfo.Status, renderErrorHTMLFrontline(title, userMessage, string(code.URN())))
				}

				if writeErr != nil {
					logger.Error("failed to write error response", "error", writeErr.Error())
				}
			} else {
				errorType = categorizeErrorTypeFrontline("", statusCode, hasError)
			}

			duration := time.Since(startTime).Seconds()
			statusStr := strconv.Itoa(statusCode)

			// Add final status to span
			span.SetAttributes(
				attribute.Int("status_code", statusCode),
				attribute.String("error_type", errorType),
			)

			logger.Info("frontline request",
				"status_code", statusStr,
				"error_type", errorType,
				"duration_seconds", duration,
				"region", region,
			)

			frontlineRequestsTotal.WithLabelValues(statusStr, errorType, region).Inc()
			frontlineRequestDuration.WithLabelValues(statusStr, errorType, region).Observe(duration)

			return nil
		}
	}
}

func getErrorPageInfoFrontline(urn codes.URN) errorPageInfo {
	//nolint:exhaustive
	switch urn {
	case codes.User.BadRequest.ClientClosedRequest.URN():
		return errorPageInfo{
			Status:  499,
			Title:   "Client Closed Request",
			Message: "The client closed the connection before the request completed.",
		}
	case codes.Frontline.Routing.ConfigNotFound.URN():
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Title:   http.StatusText(http.StatusNotFound),
			Message: "No deployment found for this hostname. Please check your domain configuration or contact support.",
		}
	case codes.Frontline.Proxy.BadGateway.URN(),
		codes.Frontline.Proxy.ProxyForwardFailed.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Title:   http.StatusText(http.StatusBadGateway),
			Message: "Unable to connect. Please try again in a few moments.",
		}
	case codes.Frontline.Proxy.ServiceUnavailable.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Title:   http.StatusText(http.StatusServiceUnavailable),
			Message: "The service is temporarily unavailable. Please try again later.",
		}
	case codes.Frontline.Proxy.GatewayTimeout.URN():
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

func renderErrorHTMLFrontline(title, message, errorCode string) []byte {
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
