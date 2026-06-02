package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/metrics"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/publicerr"
	"go.opentelemetry.io/otel/attribute"
)

// ProblemResponse is the JSON envelope returned on error. It follows
// RFC 9457 (Problem Details for HTTP APIs) with one frontline extension:
//
//   - type, title, status, detail, instance are RFC 9457 standard
//     members. `type` is a stable docs URL. `instance` is the URN
//     `urn:unkey:request:<id>` identifying the specific occurrence;
//     RFC 9457 §3.1.5 specifies a URI reference here. The bare
//     request ID is also on the `X-Unkey-Request-Id` header for
//     callers that prefer it.
//
//   - code is the only extension member: a stable public identifier
//     for the problem type, more convenient than parsing `type`.
//
// Who actually reads this body:
//
// Frontline synthesizes this response for gateway-level failures (auth,
// no route, dial failure) — it never comes from the customer's app. So a
// client SDK *generated from the customer's OpenAPI spec* has no schema
// for it and cannot consume these fields; it will fail to deserialize or
// ignore the body. The realistic consumers are humans, logs, and clients
// explicitly written to understand Unkey gateway errors (opt-in: they
// have to know our shape). This is the same model as a CDN returning its
// own error page when an origin is down.
//
// Because of that, machine-actionable signals live where a spec-agnostic
// client can still see them: the HTTP status code and the standard
// `Retry-After` header (set in WithObservability from the catalog). We
// deliberately do NOT put retry hints in the body — they'd only be
// readable by a client that already knows our schema, which by then can
// just read the header. We keep `instance` in the body anyway because a
// request ID is useful to a human reading a logged body.
//
// Per RFC 9457 the body is identical regardless of whether the client
// sent `Accept: application/json` or `application/problem+json`; only
// the Content-Type differs (handled by zen.Session.ProblemJSON).
type ProblemResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
	Code     string `json:"code"`
}

// instanceURN formats a request ID as the URN we put in the
// `instance` field. Kept here so tests can use the same formatter
// without re-deriving the prefix.
func instanceURN(requestID string) string {
	return "urn:unkey:request:" + requestID
}

// WithObservability is the request-level observability middleware. It owns:
//
//   - tracing span for the request
//   - error rendering (HTML page or JSON, based on Accept header)
//   - emission of unkey_frontline_requests_total
//   - emission of unkey_frontline_overhead_seconds (total minus upstream
//     call duration read from RequestTracking, when available)
//
// Per-component latency lives on the package that owns the work:
// routing in router, upstream timing in proxy.
func WithObservability(renderer errorpage.Renderer) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			metrics.InflightRequests.Inc()
			defer metrics.InflightRequests.Dec()

			start := time.Now()

			// Set X-Unkey-Request-Id on every response so callers can
			// correlate. The proxy sets it again on successful forwards,
			// which is harmless (same value). Done here so error paths
			// that never reach the proxy (auth fail, no route, etc.) still
			// carry it.
			s.AddHeader("X-Unkey-Request-Id", s.RequestID())

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

				problem := publicerr.ProblemFor(urn)
				statusCode = problem.Status.Int()

				// Per-request override: if the fault carries a user-facing
				// message (e.g. validation detail), prefer it over the
				// catalog default.
				detail := problem.Detail
				if msg := fault.UserFacingMessage(err); msg != "" {
					detail = msg
				}

				if statusCode == http.StatusInternalServerError {
					logger.Error("frontline error",
						"error", err.Error(),
						"requestId", s.RequestID(),
						"publicMessage", detail,
						"status", statusCode,
						"internalCode", string(urn),
						"publicCode", problem.Code,
						"path", s.Request().URL.Path,
						"host", s.Request().Host,
					)
				}

				if problem.RetryAfter != nil {
					s.AddHeader("Retry-After", strconv.Itoa(*problem.RetryAfter))
				}

				body := ProblemResponse{
					Type:     problem.TypeURL,
					Title:    problem.Title,
					Status:   statusCode,
					Detail:   detail,
					Instance: instanceURN(s.RequestID()),
					Code:     problem.Code,
				}

				writeErr := writeErrorResponse(s, renderer, problem, body)
				if writeErr != nil {
					if isClientGone(writeErr) {
						// Client disconnected before we could flush the
						// error page.
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

			outcome := metrics.OutcomeFor(urn)
			metrics.RequestsTotal.WithLabelValues(
				metrics.StatusClass(statusCode),
				string(urn),
				string(outcome),
			).Inc()

			// Frontline overhead: total handler time minus the upstream
			// call(s). Upstream duration is read from RequestTracking's
			// UpstreamDuration, which the proxy accumulates across every
			// attempt (including failed dials in the local retry loop) on
			// both local-instance and cross-region paths. If tracking is
			// absent or never reached the proxy (pre-routing failure),
			// UpstreamDuration is zero and overhead == total — correct.
			overhead := time.Since(start)
			if t, ok := proxy.RequestTrackingFromContext(ctx); ok && t.UpstreamDuration > 0 {
				overhead -= t.UpstreamDuration
			}
			if overhead < 0 {
				overhead = 0
			}
			metrics.OverheadSeconds.WithLabelValues(string(outcome)).Observe(overhead.Seconds())

			return nil
		}
	}
}

// writeErrorResponse picks the wire format based on the request
// protocol, then on the Accept header. Protocols (most specific
// first):
//
//   - gRPC (Content-Type application/grpc*): trailers-only response
//     with grpc-status / grpc-message. HTTP wire status is 200; the
//     logical status in metrics still reflects problem.Status.
//   - Connect-streaming (application/connect+*): HTTP 200 with a
//     single end-stream envelope frame carrying the error.
//   - Connect-unary (Connect-Protocol-Version header set): HTTP
//     status from Connect's code→status map, JSON body {code,message}.
//   - HTML: Accept includes text/html.
//   - JSON: everything else, via zen.Session.ProblemJSON. Picks
//     application/problem+json vs application/json based on Accept.
//
// On HTML render failure we fall back to JSON so the caller still
// gets a parseable response.
func writeErrorResponse(s *zen.Session, renderer errorpage.Renderer, problem publicerr.Problem, body ProblemResponse) error {
	switch detectProtocol(s.Request()) {
	case protocolGRPC:
		return writeGRPCError(s, problem, body.Detail)
	case protocolConnectStream:
		return writeConnectStreamError(s, problem, body.Detail)
	case protocolConnectUnary:
		return writeConnectUnaryError(s, problem, body.Detail)
	case protocolHTTP:
		// fall through
	}

	if prefersHTML(s.Request().Header.Get("Accept")) {
		htmlBody, renderErr := renderer.Render(errorpage.Data{
			StatusCode: body.Status,
			Title:      problem.Title,
			Message:    body.Detail,
			ErrorCode:  problem.Code,
			DocsURL:    problem.TypeURL,
			RequestID:  s.RequestID(),
		})
		if renderErr != nil {
			logger.Error("failed to render error page", "error", renderErr.Error())
			return s.ProblemJSON(body.Status, body)
		}
		return s.HTML(body.Status, htmlBody)
	}
	return s.ProblemJSON(body.Status, body)
}

// prefersHTML reports whether the caller asked for HTML. We treat
// any Accept that names text/html as wanting HTML; everything else
// (application/json, application/problem+json, application/*, */*,
// empty) gets JSON. Mirrors the previous JSON-first default for
// API clients (curl, SDKs) while keeping browsers on HTML.
func prefersHTML(accept string) bool {
	return strings.Contains(accept, "text/html")
}
