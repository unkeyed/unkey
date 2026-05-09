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
			span.SetAttributes(
				attribute.String("request_id", s.RequestID()),
				attribute.String("host", s.Request().Host),
				attribute.String("method", s.Request().Method),
				attribute.String("path", s.Request().URL.Path),
			)
			defer span.End()

			err := next(ctx, s)

			statusCode := s.StatusCode()
			hasError := err != nil
			var urn codes.URN

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

				userMessage := pageInfo.Message
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
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
					htmlBody, renderErr := renderer.Render(errorpage.Data{
						StatusCode: pageInfo.Status,
						Title:      pageInfo.Title,
						Message:    userMessage,
						ErrorCode:  string(code.URN()),
						DocsURL:    code.DocsURL(),
						RequestID:  s.RequestID(),
					})
					if renderErr != nil {
						logger.Error("failed to render error page", "error", renderErr.Error())
						writeErr = s.JSON(pageInfo.Status, ErrorResponse{
							Error: ErrorDetail{
								Code:    string(code.URN()),
								Message: userMessage,
							},
						})
					} else {
						writeErr = s.HTML(pageInfo.Status, htmlBody)
					}
				}

				if writeErr != nil {
					if isClientGone(writeErr) {
						// Client disconnected before we could flush the
						// response. Reclassify so metrics, traces, and
						// the 500-error log below all reflect what
						// actually happened — the client never received
						// our response, so labelling this with the
						// original error's status is misleading.
						urn = codes.User.BadRequest.ClientClosedRequest.URN()
						statusCode = codes.StatusClientClosedRequest.Int()
						logger.Debug("client gone before error response was written",
							"error", writeErr.Error(),
							"requestId", s.RequestID(),
						)
					} else {
						logger.Error("failed to write error response", "error", writeErr.Error())
					}
				}

				// Server-side faults log loudly so they page on-call.
				// Done after the write attempt so client-gone
				// reclassification suppresses this — a 500 the client
				// never received isn't actionable.
				if statusCode == http.StatusInternalServerError {
					logger.Error("frontline error",
						"error", err.Error(),
						"requestId", s.RequestID(),
						"publicMessage", userMessage,
						"status", statusCode,
						"path", s.Request().URL.Path,
						"host", s.Request().Host,
					)
				}
			}

			span.SetAttributes(
				attribute.Int("status_code", statusCode),
				attribute.String("code", string(urn)),
			)

			logger.Info("frontline request",
				"status_code", statusCode,
				"code", string(urn),
			)

			metrics.RequestsTotal.WithLabelValues(metrics.StatusClass(statusCode), string(urn)).Inc()

			return nil
		}
	}
}

// frontlineErrorMessages maps URNs to user-facing messages emitted by
// frontline. An empty string (or absent entry) means: fall back to
// fault.UserFacingMessage(err).
//
// Status and title are NOT in this map — status comes from
// httpStatus(urn) (which derives from pkg/codes), and title comes from
// errorTitle(status, urn). This map exists only to phrase product-specific
// guidance ("contact support@unkey.com", "check your domain configuration",
// etc.) that pkg/codes shouldn't know about.
var frontlineErrorMessages = map[codes.URN]string{
	codes.User.BadRequest.ClientClosedRequest.URN():         "The client closed the connection before the request completed.",
	codes.User.BadRequest.RequestTimeout.URN():              "The request took too long to process. Please try again later.",
	codes.User.BadRequest.RequestBodyTooLarge.URN():         "The request body exceeds the maximum allowed size.",
	codes.User.BadRequest.RequestBodyUnreadable.URN():       "The request body could not be read.",
	codes.Auth.Authentication.Missing.URN():                 "Authentication required.",
	codes.Auth.Authentication.Malformed.URN():               "The authentication credentials are malformed.",
	codes.Frontline.Routing.ConfigNotFound.URN():            "No deployment found for this hostname. Please check your domain configuration or contact support at support@unkey.com.",
	codes.Frontline.Routing.DeploymentNotFound.URN():        "The requested deployment could not be found.",
	codes.Frontline.Routing.NoRunningInstances.URN():        "No running instances are available to handle this request.",
	codes.Frontline.Routing.DeploymentSelectionFailed.URN(): "Failed to select an instance to handle your request.",
	codes.Frontline.Proxy.BadGateway.URN():                  "Unable to connect. Please try again in a few moments.",
	codes.Frontline.Proxy.ProxyForwardFailed.URN():          "Unable to connect. Please try again in a few moments.",
	codes.Frontline.Proxy.ServiceUnavailable.URN():          "The service is temporarily unavailable. Please try again later.",
	codes.Frontline.Proxy.GatewayTimeout.URN():              "The request took too long to process. Please try again later.",
	codes.Frontline.Auth.MissingCredentials.URN():           "Authentication required. Please provide a valid API key.",
	codes.Frontline.Auth.InvalidKey.URN():                   "Authentication failed. The provided API key is invalid.",
	codes.Frontline.Auth.InsufficientPermissions.URN():      "Access denied. The API key does not have the required permissions.",
	codes.Frontline.Auth.RateLimited.URN():                  "Rate limit exceeded. Please try again later.",
	codes.Frontline.Firewall.Denied.URN():                   "Forbidden",
	codes.Frontline.Internal.InvalidConfiguration.URN():     "The deployment configuration is invalid. Please contact support at support@unkey.com.",
	codes.Frontline.Internal.InternalServerError.URN():      "An unexpected error occurred. Please try again later.",
}

func getErrorPageInfoFrontline(urn codes.URN) errorPageInfo {
	status := httpStatus(urn)
	return errorPageInfo{
		Status:  status.Int(),
		Title:   errorTitle(status, urn),
		Message: frontlineErrorMessages[urn],
	}
}

// errorTitle returns the human-readable title for the error page. Defaults
// to status.Text() (the standard reason phrase); special-cases the few
// statuses whose stdlib name is missing or wrong for our purposes.
func errorTitle(status codes.HTTPStatus, urn codes.URN) string {
	if urn == codes.User.BadRequest.ClientClosedRequest.URN() {
		// 499 has no stdlib reason phrase.
		return "Client Closed Request"
	}
	return status.Text()
}
