package middleware

import (
	"context"
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

				status := httpStatus(urn)
				statusCode = status.Int()
				title := errorTitle(status, urn)

				userMessage := frontlineErrorMessages[urn]
				if userMessage == "" {
					userMessage = fault.UserFacingMessage(err)
				}

				if status == codes.StatusInternalServerError {
					logger.Error("frontline error",
						"error", err.Error(),
						"requestId", s.RequestID(),
						"publicMessage", userMessage,
						"status", statusCode,
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
					writeErr = s.JSON(statusCode, ErrorResponse{
						Meta: ErrorMeta{RequestID: s.RequestID()},
						Error: ErrorDetail{
							Code:    string(code.URN()),
							Message: userMessage,
						},
					})
				} else {
					htmlBody, renderErr := renderer.Render(errorpage.Data{
						StatusCode: statusCode,
						Title:      title,
						Message:    userMessage,
						ErrorCode:  string(code.URN()),
						DocsURL:    code.DocsURL(),
						RequestID:  s.RequestID(),
					})
					if renderErr != nil {
						logger.Error("failed to render error page", "error", renderErr.Error())
						writeErr = s.JSON(statusCode, ErrorResponse{
							Meta: ErrorMeta{RequestID: s.RequestID()},
							Error: ErrorDetail{
								Code:    string(code.URN()),
								Message: userMessage,
							},
						})
					} else {
						writeErr = s.HTML(statusCode, htmlBody)
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
