package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/metrics"
	"go.opentelemetry.io/otel/attribute"
)

type ErrorResponse struct {
	Meta  ErrorMeta   `json:"meta"`
	Error ErrorDetail `json:"error"`
}

type ErrorMeta struct {
	RequestID string `json:"requestId"`
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

// WithObservability is the request-level observability middleware. It owns:
//
//   - tracing span for the request
//   - error rendering (HTML page or JSON, based on Accept header)
//   - emission of unkey_frontline_requests_total
//
// Per-component latency lives on the package that owns the work: routing
// in router, upstream timing in proxy. There is no platform-overhead
// histogram here — alert on routing/upstream latency separately.
func WithObservability(renderer errorpage.Renderer) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			metrics.InflightRequests.Inc()
			defer metrics.InflightRequests.Dec()

			ctx, span := tracing.Start(ctx, "frontline.proxy")
			// Building attributes allocates; skip it for spans that were
			// sampled out (or when tracing is disabled) since they discard
			// attributes anyway.
			recording := span.IsRecording()
			if recording {
				span.SetAttributes(
					attribute.String("request_id", s.RequestID()),
					attribute.String("host", s.Request().Host),
					attribute.String("method", s.Request().Method),
					attribute.String("path", s.Request().URL.Path),
				)
			}
			defer span.End()

			err := next(ctx, s)

			statusCode := s.StatusCode()
			hasError := err != nil
			var urn codes.URN
			// fault_domain is "" on success and the attribution domain on error.
			var domain string

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

				// Attribution domain for the metric: an explicit fault.Category
				// override if set, else the category segment of the URN. Errors
				// that carry neither are unattributed — default to platform so
				// they ring our own alert instead of vanishing.
				if cat, hasCat := fault.GetCategory(err); hasCat {
					domain = string(cat)
				} else {
					domain = string(codes.CategoryPlatform)
				}

				pageInfo := getErrorPageInfoFrontline(urn)
				statusCode = pageInfo.Status

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				// Log the precise URN for every error so it can be correlated
				// with the response (which also carries the URN). Server faults
				// are logged at error level, everything else at info.
				logArgs := []any{
					"error", err.Error(),
					"requestId", s.RequestID(),
					"code", string(urn),
					"faultDomain", domain,
					"status", pageInfo.Status,
					"path", s.Request().URL.Path,
					"host", s.Request().Host,
				}
				if pageInfo.Status >= http.StatusInternalServerError {
					logger.Error("frontline error", logArgs...)
				} else {
					logger.Info("frontline request error", logArgs...)
				}

				acceptHeader := s.Request().Header.Get("Accept")
				preferJSON := strings.Contains(acceptHeader, "application/json") ||
					strings.Contains(acceptHeader, "application/*") ||
					(strings.Contains(acceptHeader, "*/*") && !strings.Contains(acceptHeader, "text/html"))

				// Surface the full URN and its docs link to the caller: it's
				// the exact string support needs for correlation, and it's
				// already logged above under the same request ID.
				var writeErr error
				if preferJSON {
					writeErr = s.JSON(pageInfo.Status, ErrorResponse{
						Meta: ErrorMeta{RequestID: s.RequestID()},
						Error: ErrorDetail{
							Code:    string(urn),
							Message: userMessage,
						},
					})
				} else {
					htmlBody, renderErr := renderer.Render(errorpage.Data{
						StatusCode: pageInfo.Status,
						Title:      pageInfo.Title,
						Message:    userMessage,
						ErrorCode:  string(urn),
						DocsURL:    code.DocsURL(),
						RequestID:  s.RequestID(),
					})
					if renderErr != nil {
						logger.Error("failed to render error page", "error", renderErr.Error())
						writeErr = s.JSON(pageInfo.Status, ErrorResponse{
							Meta: ErrorMeta{RequestID: s.RequestID()},
							Error: ErrorDetail{
								Code:    string(urn),
								Message: userMessage,
							},
						})
					} else {
						writeErr = s.HTML(pageInfo.Status, htmlBody)
					}
				}

				if writeErr != nil {
					if isClientGone(writeErr) {
						// Client disconnected before we could flush the error
						// page. Not actionable — don't alert on it.
						logger.Debug("client gone before error response was written",
							"error", writeErr.Error(),
							"requestId", s.RequestID(),
						)
					} else {
						logger.Error("failed to write error response", "error", writeErr.Error())
					}
				}
			}

			if recording {
				span.SetAttributes(
					attribute.Int("status_code", statusCode),
					attribute.String("code", string(urn)),
					attribute.String("fault_domain", domain),
				)
			}

			metrics.RequestsTotal.WithLabelValues(metrics.StatusClass(statusCode), domain, string(urn)).Inc()

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
	case codes.User.BadRequest.RequestTimeout.URN():
		// Frontline acts as a gateway: a request-processing deadline that
		// fires here means upstream took too long. Surface as 504, not 500,
		// so it doesn't trigger server-error alerting.
		return errorPageInfo{
			Status:  http.StatusGatewayTimeout,
			Title:   http.StatusText(http.StatusGatewayTimeout),
			Message: "The request took too long to process. Please try again later.",
		}
	case codes.User.BadRequest.RequestBodyTooLarge.URN():
		return errorPageInfo{
			Status:  http.StatusRequestEntityTooLarge,
			Title:   http.StatusText(http.StatusRequestEntityTooLarge),
			Message: "The request body exceeds the maximum allowed size.",
		}
	case codes.User.BadRequest.RequestBodyUnreadable.URN():
		return errorPageInfo{
			Status:  http.StatusBadRequest,
			Title:   http.StatusText(http.StatusBadRequest),
			Message: "The request body could not be read.",
		}
	case codes.Auth.Authentication.Missing.URN():
		return errorPageInfo{
			Status:  http.StatusUnauthorized,
			Title:   http.StatusText(http.StatusUnauthorized),
			Message: "Authentication required.",
		}
	case codes.Auth.Authentication.Malformed.URN():
		return errorPageInfo{
			Status:  http.StatusUnauthorized,
			Title:   http.StatusText(http.StatusUnauthorized),
			Message: "The authentication credentials are malformed.",
		}
	case codes.App.Validation.InvalidInput.URN():
		return errorPageInfo{
			Status:  http.StatusBadRequest,
			Title:   http.StatusText(http.StatusBadRequest),
			Message: "",
		}
	case codes.Frontline.Routing.ConfigNotFound.URN():
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Title:   http.StatusText(http.StatusNotFound),
			Message: "No deployment found for this hostname. Please check your domain configuration or contact support at support@unkey.com.",
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

	// Engine errors — keyauth / firewall denials produced in-process.
	// Status comes from the code's HTTP semantics (CategoryUnauthorized → 401,
	// CategoryForbidden → 403, CategoryRateLimited → 429), not from upstream.
	case codes.Frontline.Auth.MissingCredentials.URN():
		return errorPageInfo{
			Status:  http.StatusUnauthorized,
			Title:   http.StatusText(http.StatusUnauthorized),
			Message: "Authentication required. Please provide a valid API key.",
		}
	case codes.Frontline.Auth.InvalidKey.URN():
		return errorPageInfo{
			Status:  http.StatusUnauthorized,
			Title:   http.StatusText(http.StatusUnauthorized),
			Message: "Authentication failed. The provided API key is invalid.",
		}
	case codes.Frontline.Auth.InsufficientPermissions.URN():
		return errorPageInfo{
			Status:  http.StatusForbidden,
			Title:   http.StatusText(http.StatusForbidden),
			Message: "Access denied. The API key does not have the required permissions.",
		}
	case codes.Frontline.Auth.RateLimited.URN():
		return errorPageInfo{
			Status:  http.StatusTooManyRequests,
			Title:   http.StatusText(http.StatusTooManyRequests),
			Message: "Rate limit exceeded. Please try again later.",
		}
	case codes.Frontline.Firewall.Denied.URN():
		return errorPageInfo{
			Status:  http.StatusForbidden,
			Title:   http.StatusText(http.StatusForbidden),
			Message: "Forbidden",
		}
	case codes.Frontline.OpenApi.InvalidRequest.URN():
		return errorPageInfo{
			Status:  http.StatusBadRequest,
			Title:   http.StatusText(http.StatusBadRequest),
			Message: "",
		}

	// Routing failures other than ConfigNotFound (e.g. deployment-by-id miss,
	// no instances in any region).
	case codes.Frontline.Routing.DeploymentNotFound.URN():
		return errorPageInfo{
			Status:  http.StatusNotFound,
			Title:   http.StatusText(http.StatusNotFound),
			Message: "The requested deployment could not be found.",
		}
	case codes.Frontline.Routing.NoRunningInstances.URN():
		return errorPageInfo{
			Status:  http.StatusServiceUnavailable,
			Title:   http.StatusText(http.StatusServiceUnavailable),
			Message: "No running instances are available to handle this request.",
		}
	case codes.Frontline.Routing.DeploymentSelectionFailed.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Title:   http.StatusText(http.StatusInternalServerError),
			Message: "Failed to select an instance to handle your request.",
		}
	case codes.Frontline.Internal.InvalidConfiguration.URN():
		// The deployment's own policy config could not be parsed. That is the
		// config author's problem, not a frontline fault — surface as 422, not
		// 500, so it doesn't trigger server-error alerting.
		return errorPageInfo{
			Status:  http.StatusUnprocessableEntity,
			Title:   http.StatusText(http.StatusUnprocessableEntity),
			Message: "The deployment configuration is invalid. Please check your config or contact support at support@unkey.com.",
		}
	case codes.Frontline.Internal.InternalServerError.URN():
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Title:   http.StatusText(http.StatusInternalServerError),
			Message: "An unexpected error occurred. Please try again later.",
		}
	default:
		return errorPageInfo{
			Status:  http.StatusInternalServerError,
			Title:   http.StatusText(http.StatusInternalServerError),
			Message: "",
		}
	}
}
