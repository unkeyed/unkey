package middleware

import (
	"context"
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
	Status  int    `json:"status"`
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
				urn = codes.Gateway.Internal.InternalServerError.URN()
			}

			code, parseErr := codes.ParseURN(urn)
			if parseErr != nil {
				logger.Error("failed to parse error code", "error", parseErr.Error())
				code = codes.Gateway.Internal.InternalServerError
			}

			// Determine status code based on error type
			status := http.StatusInternalServerError
			switch urn {
			// Internal Server Error (500)
			case codes.UnkeyGatewayErrorsInternalInternalServerError,
				codes.UnkeyGatewayErrorsInternalKeyVerificationFailed,
				codes.UnkeyGatewayErrorsRoutingVMSelectionFailed:
				status = http.StatusInternalServerError

			// Bad Request (400)
			case codes.UnkeyGatewayErrorsValidationRequestInvalid,
				codes.UnkeyGatewayErrorsValidationResponseInvalid:
				status = http.StatusBadRequest

			// Unauthorized (401)
			case codes.UnkeyGatewayErrorsAuthUnauthorized:
				status = http.StatusUnauthorized

			// Not Found (404)
			case codes.UnkeyGatewayErrorsRoutingConfigNotFound:
				status = http.StatusNotFound

			// Too Many Requests (429)
			case codes.UnkeyGatewayErrorsAuthRateLimited:
				status = http.StatusTooManyRequests

			// Bad Gateway (502)
			case codes.UnkeyGatewayErrorsProxyBadGateway,
				codes.UnkeyGatewayErrorsProxyProxyForwardFailed:
				status = http.StatusBadGateway

			// Service Unavailable (503)
			case codes.UnkeyGatewayErrorsProxyServiceUnavailable:
				status = http.StatusServiceUnavailable

			// Gateway Timeout (504)
			case codes.UnkeyGatewayErrorsProxyGatewayTimeout:
				status = http.StatusGatewayTimeout
			}

			// Log the error with correct status
			logger.Error("gate error",
				"error", err.Error(),
				"requestId", s.RequestID(),
				"publicMessage", fault.UserFacingMessage(err),
				"status", status,
				"path", s.Request().URL.Path,
				"host", s.Request().Host,
			)

			// Determine response format based on Accept header
			acceptHeader := s.Request().Header.Get("Accept")

			// Prefer JSON if:
			// 1. Accept explicitly includes application/json
			// 2. Accept includes application/* (wildcard)
			// 3. Accept is */* but does NOT include text/html (API clients)
			preferJSON := strings.Contains(acceptHeader, "application/json") ||
				strings.Contains(acceptHeader, "application/*") ||
				(strings.Contains(acceptHeader, "*/*") && !strings.Contains(acceptHeader, "text/html"))

			// If client prefers JSON, return JSON error
			if preferJSON {
				return s.JSON(status, ErrorResponse{
					Error: ErrorDetail{
						Code:    string(code.URN()),
						Message: fault.UserFacingMessage(err),
						Status:  status,
					},
				})
			}

			// Otherwise return HTML error pages for customer-facing errors (browsers)
			//nolint:exhaustive
			switch urn {
			case codes.UnkeyGatewayErrorsRoutingConfigNotFound:
				return s.HTML(http.StatusNotFound, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>404 Not Found</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>404 Not Found</h1>
    <p>No deployment found for this hostname.</p>
    <p>Please check your domain configuration or contact support.</p>
    <p class="error-code">Error: `+string(code.URN())+`</p>
</body>
</html>`))

			case codes.UnkeyGatewayErrorsProxyBadGateway,
				codes.UnkeyGatewayErrorsProxyProxyForwardFailed:
				return s.HTML(http.StatusBadGateway, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>502 Bad Gateway</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>502 Bad Gateway</h1>
    <p>Unable to connect to the backend service.</p>
    <p>Please try again in a few moments.</p>
    <p class="error-code">Error: `+string(code.URN())+`</p>
</body>
</html>`))

			case codes.UnkeyGatewayErrorsProxyServiceUnavailable:
				return s.HTML(http.StatusServiceUnavailable, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>503 Service Unavailable</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>503 Service Unavailable</h1>
    <p>The service is temporarily unavailable.</p>
    <p>Please try again later.</p>
    <p class="error-code">Error: `+string(code.URN())+`</p>
</body>
</html>`))

			case codes.UnkeyGatewayErrorsProxyGatewayTimeout:
				return s.HTML(http.StatusGatewayTimeout, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>504 Gateway Timeout</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>504 Gateway Timeout</h1>
    <p>The request took too long to process.</p>
    <p>Please try again later.</p>
    <p class="error-code">Error: `+string(code.URN())+`</p>
</body>
</html>`))

			default:
				// For any other errors, return a generic error page
				return s.HTML(status, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>`+http.StatusText(status)+`</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>`+http.StatusText(status)+`</h1>
    <p>`+fault.UserFacingMessage(err)+`</p>
    <p class="error-code">Error: `+string(code.URN())+`</p>
</body>
</html>`))
			}
		}
	}
}
