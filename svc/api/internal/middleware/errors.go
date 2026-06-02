package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// apiStatusOverrides holds the cases where the api emits a different HTTP
// status than the code's canonical mapping in pkg/codes.
//
// The api treats missing or malformed credentials as 400 Bad Request rather
// than 401 Unauthorized. This is historical behavior, kept for backwards
// compatibility with documented client expectations.
//
//nolint:exhaustive // sparse override map; not every URN has an override.
var apiStatusOverrides = map[codes.URN]codes.HTTPStatus{
	codes.Auth.Authentication.Missing.URN():   codes.StatusBadRequest,
	codes.Auth.Authentication.Malformed.URN(): codes.StatusBadRequest,
	codes.Portal.Session.TokenMissing.URN():   codes.StatusBadRequest,
}

// apiTitleOverrides holds per-URN titles that differ from the standard reason
// phrase (HTTPStatus.Text()). Most error pages use the stdlib reason phrase;
// entries here express api-specific phrasing.
//
//nolint:exhaustive // sparse override map; not every URN has an override.
var apiTitleOverrides = map[codes.URN]string{
	codes.User.BadRequest.ClientClosedRequest.URN():        "Client Closed Request", // 499 has no stdlib name
	codes.Data.RatelimitNamespace.Gone.URN():               "Resource Gone",
	codes.Auth.Authorization.InsufficientPermissions.URN(): "Insufficient Permissions",
	codes.Data.Permission.Duplicate.URN():                  "Conflicting Resource",
	codes.Data.Role.Duplicate.URN():                        "Conflicting Resource",
	codes.Data.Identity.Duplicate.URN():                    "Conflicting Resource",
	codes.App.Protection.ProtectedResource.URN():           "Resource is protected",
	codes.Auth.Authorization.KeyDisabled.URN():             "Key is disabled",
	codes.Auth.Authorization.WorkspaceDisabled.URN():       "Workspace is disabled",
}

// httpStatus resolves the HTTP status the api should return for urn.
// Resolution order:
//
//  1. Per-service override (apiStatusOverrides).
//  2. Per-code mapping in pkg/codes (Code.HTTPStatus()).
//  3. 500, when the URN is malformed or the code lives in a non-HTTP category
//     and leaked into a request path. Both are bugs worth alerting on.
func httpStatus(urn codes.URN) codes.HTTPStatus {
	if s, ok := apiStatusOverrides[urn]; ok {
		return s
	}

	code, err := codes.ParseURN(urn)
	if err != nil {
		return codes.StatusInternalServerError
	}

	if s := code.HTTPStatus(); s != 0 {
		return s
	}

	return codes.StatusInternalServerError
}

// errorTitle returns the human-readable title for the response. It defaults to
// the standard reason phrase (HTTPStatus.Text()); special cases live in
// apiTitleOverrides.
func errorTitle(status codes.HTTPStatus, urn codes.URN) string {
	if t, ok := apiTitleOverrides[urn]; ok {
		return t
	}
	return status.Text()
}

// errorLogAttrs builds the structured attributes attached to every 5xx log
// from this middleware. The goal is to give an on-call engineer enough
// context to find the failing request in ClickHouse / wide-event logs
// without having to grep around: workspace and request id pin the row,
// http.* describes what was called, error.* (added by the logger's fault
// handler) describes what went wrong.
//
// We deliberately don't include the request body: it can contain secrets
// and is already redacted-and-logged by the wide-event request logger.
// Key id / identity id are not available here (they live on KeyVerifier,
// not zen.Session), but the request id lets you pivot to the wide event
// that does carry them.
func errorLogAttrs(s *zen.Session, err error, status int, urn codes.URN) []any {
	return []any{
		"error", err,
		"workspaceId", s.AuthorizedWorkspaceID(),
		"requestId", s.RequestID(),
		"code", string(urn),
		"publicMessage", fault.UserFacingMessage(err),
		slog.Group("http",
			slog.String("method", s.Request().Method),
			slog.String("path", s.Request().URL.Path),
			slog.String("query", s.Request().URL.RawQuery),
			slog.String("host", s.Request().Host),
			slog.String("user_agent", s.UserAgent()),
			slog.String("ip", s.Location()),
			slog.String("referer", s.Request().Referer()),
			slog.Int("status", status),
		),
	}
}

// WithErrorHandling returns middleware that translates errors into appropriate
// HTTP responses based on the URN attached to the error.
//
// The status comes from pkg/codes (Category default plus per-code overrides)
// layered with the per-service overrides above. The response body shape still
// varies per status (BadRequestErrorResponse, NotFoundErrorResponse, ...) and
// is selected by buildErrorBody, which switches on status rather than URN, so
// adding a new code never requires touching this file.
func WithErrorHandling() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			err := next(ctx, s)
			if err == nil {
				return nil
			}

			// Store the internal error message for metrics logging before we
			// convert it to an HTTP response and lose the details.
			s.SetInternalError(fault.InternalMessage(err))

			urn, ok := fault.GetCode(err)
			if !ok {
				urn = codes.App.Internal.UnexpectedError.URN()
			}

			code, parseErr := codes.ParseURN(urn)
			if parseErr != nil {
				logger.Error("failed to parse error code", "error", parseErr.Error())
				code = codes.App.Internal.UnexpectedError
				urn = code.URN()
			}

			status := httpStatus(urn)

			// The api's openapi schema has no 502/504 response shapes —
			// those statuses only originate from ingress/frontline codes,
			// not from the api itself. Collapse them to 500 here so the
			// HTTP status line, the response body, and the on-call log
			// guard below all agree (buildErrorBody also forces the body to
			// 500, but the written status must match it).
			if status == codes.StatusBadGateway || status == codes.StatusGatewayTimeout {
				status = codes.StatusInternalServerError
			}

			detail := fault.UserFacingMessage(err)
			title := errorTitle(status, urn)

			// Analytics connection failures are an operational concern even
			// though they're a 503, not a 500. Log them so on-call can see the
			// upstream is down.
			if urn == codes.Data.Analytics.ConnectionFailed.URN() {
				logger.Error(
					"analytics connection error",
					errorLogAttrs(s, err, status.Int(), urn)...,
				)
			}

			// Genuine 500s log loudly so they page on-call.
			if status == codes.StatusInternalServerError {
				logger.Error(
					"api error",
					errorLogAttrs(s, err, status.Int(), urn)...,
				)
			}

			body := buildErrorBody(status, title, code.DocsURL(), detail, s.RequestID())
			return s.ProblemJSON(status.Int(), body)
		}
	}
}

// buildErrorBody returns the openapi response body matching status. The api's
// openapi schema defines a separate response type per status
// (NotFoundErrorResponse, BadRequestErrorResponse, ...); they all share the
// {Meta, Error} shape with BaseError, except the 4xx-with-validation statuses
// which use BadRequestErrorDetails.
func buildErrorBody(status codes.HTTPStatus, title, docsURL, detail, requestID string) any {
	meta := openapi.Meta{RequestId: requestID}
	base := openapi.BaseError{
		Title:  title,
		Type:   docsURL,
		Detail: detail,
		Status: status.Int(),
	}
	badReqDetails := openapi.BadRequestErrorDetails{
		Title:  title,
		Type:   docsURL,
		Detail: detail,
		Status: status.Int(),
		Errors: []openapi.ValidationError{},
	}

	switch status {
	case codes.StatusBadRequest,
		codes.StatusRequestTimeout,
		codes.StatusRequestEntityTooLarge,
		codes.StatusClientClosedRequest:
		return openapi.BadRequestErrorResponse{Meta: meta, Error: badReqDetails}
	case codes.StatusUnauthorized:
		return openapi.UnauthorizedErrorResponse{Meta: meta, Error: base}
	case codes.StatusForbidden:
		return openapi.ForbiddenErrorResponse{Meta: meta, Error: base}
	case codes.StatusNotFound:
		return openapi.NotFoundErrorResponse{Meta: meta, Error: base}
	case codes.StatusConflict:
		return openapi.ConflictErrorResponse{Meta: meta, Error: base}
	case codes.StatusGone:
		return openapi.GoneErrorResponse{Meta: meta, Error: base}
	case codes.StatusPreconditionFailed:
		return openapi.PreconditionFailedErrorResponse{Meta: meta, Error: base}
	case codes.StatusUnprocessableEntity:
		return openapi.UnprocessableEntityErrorResponse{Meta: meta, Error: base}
	case codes.StatusTooManyRequests:
		return openapi.TooManyRequestsErrorResponse{Meta: meta, Error: base}
	case codes.StatusServiceUnavailable:
		return openapi.ServiceUnavailableErrorResponse{Meta: meta, Error: base}
	case codes.StatusInternalServerError,
		codes.StatusBadGateway,
		codes.StatusGatewayTimeout:
		base.Status = http.StatusInternalServerError
		base.Title = http.StatusText(http.StatusInternalServerError)
		return openapi.InternalServerErrorResponse{Meta: meta, Error: base}
	}

	// Unmapped status: surface as 500 so on-call notices. The exhaustive
	// linter flags any new HTTPStatus that lands in pkg/codes without a body
	// mapping here.
	base.Status = http.StatusInternalServerError
	base.Title = http.StatusText(http.StatusInternalServerError)
	return openapi.InternalServerErrorResponse{Meta: meta, Error: base}
}
