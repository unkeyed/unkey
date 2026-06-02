// Package publicerr maps frontline's internal error URNs to a small set
// of stable, customer-facing problem codes. The internal URN is for our
// own logs, metrics, traces, and support. The public code is what
// callers see in JSON error bodies and HTML error pages.
//
// The two layers exist for distinct reasons:
//
//   - The internal URN is high-fidelity ("peer_frontline_dns_timeout").
//     It exposes our topology and is allowed to change as we change the
//     implementation.
//
//   - The public code is stable and customer-actionable. It should not
//     leak topology and should not change just because we change how a
//     failure is observed internally.
//
// The mapping has two parts:
//
//  1. publicCodeFor selects the public code from a URN. Most URNs
//     collapse to their URN category (service_unavailable, bad_gateway,
//     …). Only URNs that a caller can act on differently — auth,
//     firewall, openapi validation, payload size — pass through their
//     specific part. A handful of cross-system URNs whose category
//     isn't an HTTP-like word (Auth.Authentication.* is "authentication")
//     are remapped explicitly.
//
//  2. catalog is the public code → (Title, Detail, TypeURL, RetryAfter)
//     table. The HTTP status is derived from the URN via
//     codes.Code.HTTPStatus (pkg/codes), so it is not duplicated here.
//
// RetryAfter drives the standard HTTP `Retry-After` header (set by the
// observability middleware). It is intentionally NOT placed in the
// response body: frontline error bodies are synthesized for gateway
// failures and cannot be consumed by a client SDK generated from the
// customer's own spec, so machine-actionable hints belong on the status
// code and the header, not in a body field only an Unkey-aware client
// could read.
//
// Adding an internal URN does not require touching this package: it
// will fall through to its category. To expose a new public code,
// add the URN to publicCodeFor and the public code to catalog.
package publicerr

import (
	"github.com/unkeyed/unkey/pkg/codes"
)

// Problem is the customer-facing description of a failure.
type Problem struct {
	// Code is the stable, customer-facing identifier (e.g.
	// "service_unavailable", "invalid_key"). Safe to publish, safe to
	// document, safe to depend on.
	Code string

	// Status is the HTTP status to return. Owned by the public code
	// in the catalog so a remap (e.g. internal RequestTimeout URN →
	// public gateway_timeout) carries the customer-facing status,
	// not the URN's native status.
	Status codes.HTTPStatus

	// Title is a short human-readable label for HTML error pages and
	// RFC 9457 `title`.
	Title string

	// Detail is the default human-readable message. Callers may
	// override per-request with a fault.UserFacingMessage when set.
	Detail string

	// TypeURL is the documentation URL for this public code.
	// Populates the RFC 9457 `type` field.
	TypeURL string

	// RetryAfter is the suggested wait, in seconds, before retrying.
	// Nil when the catalog does not have a default; observability
	// emits the HTTP Retry-After header when set. Not emitted in the
	// response body — see the package doc for why.
	RetryAfter *int
}

// ProblemFor returns the public Problem for an internal URN. Unknown
// URNs collapse to internal_server_error so a missing entry surfaces
// as a generic 500 rather than leaking the URN string.
func ProblemFor(urn codes.URN) Problem {
	publicCode := publicCodeFor(urn)
	entry, ok := catalog[publicCode]
	if !ok {
		entry = catalog["internal_server_error"]
		publicCode = "internal_server_error"
	}

	return Problem{
		Code:       publicCode,
		Status:     entry.status,
		Title:      entry.title,
		Detail:     entry.detail,
		TypeURL:    entry.typeURL,
		RetryAfter: entry.retryAfter,
	}
}

// publicCodeFor returns the public code for an internal URN.
//
// Default is the URN's category (service_unavailable, bad_gateway,
// gateway_timeout, not_found, internal_server_error, …). The explicit
// cases below are the two situations that need overrides:
//
//   - Actionable 4xx that the caller can branch on: pass the specific
//     part through as the public code.
//   - Cross-system URNs whose category isn't an HTTP-like word
//     (e.g. "authentication", "application"): remap to the nearest
//     HTTP class.
func publicCodeFor(urn codes.URN) string {
	//nolint:exhaustive // explicit cases override the category default; all other URNs fall through to the URN's category.
	switch urn {
	// Actionable 4xx — caller distinguishes meaning.
	case codes.Frontline.Auth.MissingCredentials.URN():
		return "missing_credentials"
	case codes.Frontline.Auth.InvalidKey.URN():
		return "invalid_key"
	case codes.Frontline.Auth.InsufficientPermissions.URN():
		return "insufficient_permissions"
	case codes.Frontline.Auth.RateLimited.URN():
		return "rate_limited"
	case codes.Frontline.Firewall.Denied.URN():
		return "firewall_denied"
	case codes.Frontline.OpenApi.InvalidRequest.URN():
		return "openapi_invalid_request"
	case codes.User.BadRequest.RequestBodyTooLarge.URN():
		return "request_body_too_large"

	// Frontline acts as a gateway: an internal request-processing
	// deadline surfaces as a gateway timeout, not a 4xx request_timeout.
	case codes.User.BadRequest.RequestTimeout.URN():
		return "gateway_timeout"

	// Client disconnected before we finished. 499 is non-standard;
	// surface as a dedicated public code so callers can recognise it.
	case codes.User.BadRequest.ClientClosedRequest.URN():
		return "client_closed_request"

	// Cross-system URNs whose category isn't an HTTP-like word.
	case codes.Auth.Authentication.Missing.URN(),
		codes.Auth.Authentication.Malformed.URN():
		return "unauthorized"
	case codes.App.Validation.InvalidInput.URN():
		return "bad_request"
	}

	c, err := codes.ParseURN(urn)
	if err != nil {
		return "internal_server_error"
	}
	return string(c.Category)
}

// catalogEntry holds the customer-facing data for a public code.
type catalogEntry struct {
	status     codes.HTTPStatus
	title      string
	detail     string
	typeURL    string
	retryAfter *int
}

// intPtr returns a pointer to v. Used for inline RetryAfter literals.
func intPtr(v int) *int { return &v }

// catalog is the public code → catalogEntry table. This is the entire
// customer-facing error surface for frontline.
var catalog = map[string]catalogEntry{
	// ── Actionable 4xx ────────────────────────────────────────────
	// Caller must fix their request; retrying as-is will not help.
	"missing_credentials": {
		status:     codes.StatusUnauthorized,
		title:      "Unauthorized",
		detail:     "Authentication required. Provide a valid API key.",
		typeURL:    docsURL("missing-credentials"),
		retryAfter: nil,
	},
	"invalid_key": {
		status:     codes.StatusUnauthorized,
		title:      "Unauthorized",
		detail:     "The provided API key is invalid, disabled, or expired.",
		typeURL:    docsURL("invalid-key"),
		retryAfter: nil,
	},
	"insufficient_permissions": {
		status:     codes.StatusForbidden,
		title:      "Forbidden",
		detail:     "The API key does not have the required permissions.",
		typeURL:    docsURL("insufficient-permissions"),
		retryAfter: nil,
	},
	"rate_limited": {
		status:     codes.StatusTooManyRequests,
		title:      "Too Many Requests",
		detail:     "Rate limit exceeded. Retry later.",
		typeURL:    docsURL("rate-limited"),
		retryAfter: intPtr(30),
	},
	"firewall_denied": {
		status:     codes.StatusForbidden,
		title:      "Forbidden",
		detail:     "The request was blocked by a firewall rule.",
		typeURL:    docsURL("firewall-denied"),
		retryAfter: nil,
	},
	"openapi_invalid_request": {
		status:     codes.StatusBadRequest,
		title:      "Bad Request",
		detail:     "The request does not match the API specification.",
		typeURL:    docsURL("openapi-invalid-request"),
		retryAfter: nil,
	},
	"request_body_too_large": {
		status:     codes.StatusRequestEntityTooLarge,
		title:      "Payload Too Large",
		detail:     "The request body exceeds the maximum allowed size.",
		typeURL:    docsURL("request-body-too-large"),
		retryAfter: nil,
	},
	"client_closed_request": {
		status:     codes.StatusClientClosedRequest,
		title:      "Client Closed Request",
		detail:     "The client closed the connection before the request completed.",
		typeURL:    docsURL("client-closed-request"),
		retryAfter: nil,
	},

	// ── Generic 4xx classes ───────────────────────────────────────
	"bad_request": {
		status:     codes.StatusBadRequest,
		title:      "Bad Request",
		detail:     "The request is invalid.",
		typeURL:    docsURL("bad-request"),
		retryAfter: nil,
	},
	"unauthorized": {
		status:     codes.StatusUnauthorized,
		title:      "Unauthorized",
		detail:     "Authentication is required.",
		typeURL:    docsURL("unauthorized"),
		retryAfter: nil,
	},
	"forbidden": {
		status:     codes.StatusForbidden,
		title:      "Forbidden",
		detail:     "Access denied.",
		typeURL:    docsURL("forbidden"),
		retryAfter: nil,
	},
	"not_found": {
		status:     codes.StatusNotFound,
		title:      "Not Found",
		detail:     "No deployment is configured for this request.",
		typeURL:    docsURL("not-found"),
		retryAfter: nil,
	},
	"too_many_requests": {
		status:     codes.StatusTooManyRequests,
		title:      "Too Many Requests",
		detail:     "Rate limit exceeded. Retry later.",
		typeURL:    docsURL("too-many-requests"),
		retryAfter: intPtr(30),
	},

	// ── 5xx classes ───────────────────────────────────────────────
	// Transient by default: the caller can retry with backoff.
	"internal_server_error": {
		status:     codes.StatusInternalServerError,
		title:      "Internal Server Error",
		detail:     "An unexpected error occurred. Retry later.",
		typeURL:    docsURL("internal-server-error"),
		retryAfter: nil,
	},
	"bad_gateway": {
		status:     codes.StatusBadGateway,
		title:      "Bad Gateway",
		detail:     "Unable to connect. Retry in a few moments.",
		typeURL:    docsURL("bad-gateway"),
		retryAfter: intPtr(15),
	},
	"service_unavailable": {
		status:     codes.StatusServiceUnavailable,
		title:      "Service Unavailable",
		detail:     "The service is temporarily unavailable. Retry later.",
		typeURL:    docsURL("service-unavailable"),
		retryAfter: intPtr(30),
	},
	"gateway_timeout": {
		status:     codes.StatusGatewayTimeout,
		title:      "Gateway Timeout",
		detail:     "The request took too long to process. Retry later.",
		typeURL:    docsURL("gateway-timeout"),
		retryAfter: intPtr(15),
	},
}

// docsURL builds the public docs URL for a public code's TypeURL.
//
// This URL is part of the API contract. RFC 9457 says `type` should
// stably identify a problem type, and customer SDKs / logs / support
// tickets will quote these URLs. Two rules follow:
//
//  1. Slug stability — once published, a slug never renames. New error
//     class → new slug. To rename, you would also have to keep the old
//     slug routing to the new docs page forever.
//  2. Path stability — `/docs/errors/<slug>` is part of the contract.
//     If the docs site reshapes, the old paths must 301-redirect to
//     the new ones. Don't change the prefix here without coordinating
//     redirects.
//
// The URL is intentionally not versioned. Errors don't version like
// APIs do, and `/v1/` here would be cosmetic fossilization. If we ever
// want to move errors off the main docs site, a vanity host (e.g.
// errors.unkey.com) redirecting to wherever the docs live is cheaper
// than a path-level version.
func docsURL(slug string) string {
	return "https://unkey.com/docs/errors/" + slug
}
