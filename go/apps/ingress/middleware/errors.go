package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
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

// errorPageInfo holds the data needed to render an error page.
type errorPageInfo struct {
	Status  int
	Title   string
	Message string
}

// getErrorPageInfo returns the HTTP status and user-friendly message for an error URN.
func getErrorPageInfo(urn codes.URN) errorPageInfo {
	switch urn {
	case codes.UnkeyIngressErrorsRoutingConfigNotFound:
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Title:   http.StatusText(http.StatusNotFound),
			Message: "No deployment found for this hostname. Please check your domain configuration or contact support.",
		}

	case codes.UnkeyIngressErrorsProxyBadGateway,
		codes.UnkeyIngressErrorsProxyProxyForwardFailed:
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Title:   http.StatusText(http.StatusBadGateway),
			Message: "Unable to connect to the backend service. Please try again in a few moments.",
		}

	case codes.UnkeyIngressErrorsProxyServiceUnavailable:
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Title:   http.StatusText(http.StatusServiceUnavailable),
			Message: "The service is temporarily unavailable. Please try again later.",
		}

	case codes.UnkeyIngressErrorsProxyGatewayTimeout:
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

// renderErrorHTML generates an HTML error page.
func renderErrorHTML(title, message, errorCode string) []byte {
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
</html>`, title, title, message, errorCode)
}

// WithErrorHandling returns middleware that translates errors into appropriate
// HTTP responses based on error URNs and content negotiation.
//
// Response format is determined by the Accept header:
// - Returns JSON when Accept includes application/json (SDKs, API clients)
// - Returns HTML for browsers (Accept includes text/html)
// - Defaults to HTML for unknown/missing Accept headers
func WithErrorHandling(logger logging.Logger) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			err := next(ctx, s)
			if err == nil {
				return nil
			}

			// Get the error URN from the error
			urn, ok := fault.GetCode(err)
			if !ok {
				urn = codes.Ingress.Internal.InternalServerError.URN()
			}

			code, parseErr := codes.ParseURN(urn)
			if parseErr != nil {
				logger.Error("failed to parse error code", "error", parseErr.Error())
				code = codes.Ingress.Internal.InternalServerError
			}

			// Get error page info (status and message) based on URN
			pageInfo := getErrorPageInfo(urn)

			// Use user-facing message from error if no specific message defined
			userMessage := pageInfo.Message
			if userMessage == "" {
				userMessage = fault.UserFacingMessage(err)
			}

			// Use status text as title if not specifically defined
			title := pageInfo.Title
			if title == http.StatusText(http.StatusInternalServerError) {
				title = http.StatusText(pageInfo.Status)
			}

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

			// Determine response format based on Accept header
			acceptHeader := s.Request().Header.Get("Accept")

			// Prefer JSON if:
			// 1. Accept explicitly includes application/json
			// 2. Accept includes application/* (wildcard)
			// 3. Accept is */* but does NOT include text/html (API clients)
			preferJSON := strings.Contains(acceptHeader, "application/json") ||
				strings.Contains(acceptHeader, "application/*") ||
				(strings.Contains(acceptHeader, "*/*") && !strings.Contains(acceptHeader, "text/html"))

			// Return JSON error for API clients
			if preferJSON {
				return s.JSON(pageInfo.Status, ErrorResponse{
					Error: ErrorDetail{
						Code:    string(code.URN()),
						Message: userMessage,
					},
				})
			}

			// Return HTML error page for browsers
			return s.HTML(pageInfo.Status, renderErrorHTML(title, userMessage, string(code.URN())))
		}
	}
}
