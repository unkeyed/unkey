package server

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ErrorResponse is the standard error response format for the gateway.
type ErrorResponse struct {
	Meta  Meta        `json:"meta"`
	Error ErrorDetail `json:"error"`
}

// Meta contains metadata about the request.
type Meta struct {
	RequestID string `json:"requestId"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Title  string `json:"title"`
	Type   string `json:"type,omitempty"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

// WithErrorHandling returns middleware that translates errors into appropriate
// HTTP responses based on error URNs.
func WithErrorHandling(logger logging.Logger) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			err := next(ctx, s)
			if err == nil {
				return nil
			}

			// Get the error URN from the error
			urn, ok := fault.GetCode(err)
			if !ok {
				urn = codes.App.Internal.UnexpectedError.URN()
			}

			code, parseErr := codes.ParseURN(urn)
			if parseErr != nil {
				logger.Error("failed to parse error code", "error", parseErr.Error())
				code = codes.App.Internal.UnexpectedError
			}

			// Determine status code based on error type
			status := http.StatusInternalServerError

			// Log the error
			logger.Error("gateway error",
				"error", err.Error(),
				"requestId", s.RequestID(),
				"publicMessage", fault.UserFacingMessage(err),
				"status", status,
			)

			switch urn {
			// Bad Request errors
			case codes.GatewayErrorsBadRequestBadGateway:
				return s.HTML(http.StatusBadGateway, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>502 Bad Gateway</title>
</head>
<body>
   <h1>502 Bad Gateway</h1>
   <p>The server received an invalid response from an upstream server.</p>
   <p>Please try again later.</p>
   <a href="/">Return to homepage</a>
</body>
</html>`))

			}

			// Create error response
			return s.JSON(status, ErrorResponse{
				Meta: Meta{
					RequestID: s.RequestID(),
				},
				Error: ErrorDetail{
					Title:  http.StatusText(status),
					Type:   code.DocsURL(),
					Detail: fault.UserFacingMessage(err),
					Status: status,
				},
			})
		}
	}
}
