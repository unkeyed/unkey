package middleware

import (
	"context"
	"net/http"

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
	case codes.Gateway.Proxy.BadGateway.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Message: "The upstream service returned an invalid response.",
		}
	case codes.Gateway.Proxy.ServiceUnavailable.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Message: "The service is temporarily unavailable.",
		}
	case codes.Gateway.Proxy.GatewayTimeout.URN():
		return errorPageInfo{
			Status:  http.StatusGatewayTimeout,
			Message: "The upstream service did not respond in time.",
		}
	case codes.Gateway.Proxy.ProxyForwardFailed.URN():
		return errorPageInfo{
			Status:  http.StatusBadGateway,
			Message: "Failed to forward your request to the service.",
		}

	// Gateway Internal Errors
	case codes.Gateway.Internal.InternalServerError.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "An unexpected error occurred.",
		}
	case codes.Gateway.Internal.InvalidConfiguration.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "The service configuration is invalid.",
		}

	// User Request Errors
	case codes.User.BadRequest.MissingRequiredHeader.URN():
		return errorPageInfo{
			Status:  http.StatusBadRequest,
			Message: "A required header is missing from your request.",
		}
	case codes.User.BadRequest.ClientClosedRequest.URN():
		return errorPageInfo{
			Status:  499, // Non-standard but widely used for client closed connection
			Message: "The client closed the connection before the request completed.",
		}

	// Default fallback
	default:
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Message: "",
		}
	}
}

// WithErrorHandling returns middleware that translates errors into appropriate
// JSON responses based on error URNs. Gateway is not user-facing - only ingress calls it.
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

			// Get error page info (status and message) based on URN
			pageInfo := getErrorPageInfo(urn)

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

			// Always return JSON (gateway is not user-facing, only ingress calls it)
			return s.JSON(pageInfo.Status, ErrorResponse{
				Error: ErrorDetail{
					Code:    string(code.URN()),
					Message: userMessage,
				},
			})
		}
	}
}
