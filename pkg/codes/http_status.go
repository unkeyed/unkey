package codes

import (
	"net/http"
)

// HTTPStatus is a typed HTTP status code. Values are in the range 100-599.
// Construct one via a StatusXxx constant; raw int conversion is discouraged
// outside this package.
//
// A dedicated type (rather than a bare int) lets signatures like
// func httpStatus(urn) HTTPStatus state intent at the call site, keeps the
// value from being confused with other ints (timeouts, ports, byte counts),
// and keeps validation in one place.
type HTTPStatus int

// Status constants for the HTTP responses we emit. The list is the union of
// statuses used across our error-response middlewares. Values are pulled from
// net/http to keep the standard library as the source of truth; 499 ("client
// closed request") is non-stdlib but widely used.
const (
	StatusBadRequest            HTTPStatus = http.StatusBadRequest
	StatusUnauthorized          HTTPStatus = http.StatusUnauthorized
	StatusForbidden             HTTPStatus = http.StatusForbidden
	StatusNotFound              HTTPStatus = http.StatusNotFound
	StatusRequestTimeout        HTTPStatus = http.StatusRequestTimeout
	StatusConflict              HTTPStatus = http.StatusConflict
	StatusRequestEntityTooLarge HTTPStatus = http.StatusRequestEntityTooLarge
	StatusGone                  HTTPStatus = http.StatusGone
	StatusPreconditionFailed    HTTPStatus = http.StatusPreconditionFailed
	StatusUnprocessableEntity   HTTPStatus = http.StatusUnprocessableEntity
	StatusTooManyRequests       HTTPStatus = http.StatusTooManyRequests
	StatusClientClosedRequest   HTTPStatus = 499
	StatusInternalServerError   HTTPStatus = http.StatusInternalServerError
	StatusBadGateway            HTTPStatus = http.StatusBadGateway
	StatusServiceUnavailable    HTTPStatus = http.StatusServiceUnavailable
	StatusGatewayTimeout        HTTPStatus = http.StatusGatewayTimeout
)

// Int returns the underlying integer for use with http.ResponseWriter.WriteHeader.
func (s HTTPStatus) Int() int { return int(s) }

// Text returns the canonical reason phrase for s ("Not Found", "Unauthorized", ...).
// It returns the empty string for non-standard codes that net/http does not
// recognise (such as 499).
func (s HTTPStatus) Text() string { return http.StatusText(int(s)) }

// HTTPStatus returns the canonical HTTP status for this category, or 0 for
// categories that are not surfaced as a single HTTP status. A zero return has
// two causes: the category is internal infrastructure that should never reach
// an HTTP boundary (CategoryUnkeyVault), or it is too broad to carry one status
// (CategoryUnkeyData), in which case the per-code override map resolves it.
func (c Category) HTTPStatus() HTTPStatus {
	switch c {
	case CategoryUserBadRequest:
		return StatusBadRequest
	case CategoryUserUnprocessableEntity:
		return StatusUnprocessableEntity
	case CategoryNotFound:
		return StatusNotFound
	case CategoryBadGateway:
		return StatusBadGateway
	case CategoryServiceUnavailable:
		return StatusServiceUnavailable
	case CategoryGatewayTimeout:
		return StatusGatewayTimeout
	case CategoryUnauthorized, CategoryUnkeyAuthentication:
		return StatusUnauthorized
	case CategoryForbidden, CategoryUnkeyAuthorization:
		return StatusForbidden
	case CategoryRateLimited, CategoryUserTooManyRequests, CategoryUnkeyLimits:
		return StatusTooManyRequests
	case CategoryInternalServerError, CategoryUnkeyApplication:
		return StatusInternalServerError
	case CategoryUnkeyData:
		// Data is too broad for a single status; individual codes resolve
		// via codeStatusOverrides (404, 409, 410, 412, 503, ...).
		return 0
	case CategoryUnkeyVault:
		// Vault is internal infrastructure; a leak to HTTP is a bug.
		return 0
	}
	return 0
}

// codeStatusOverrides maps the codes whose HTTP status is NOT implied by their
// Category back to an explicit status. Two situations land a code here:
//
//  1. It lives in CategoryUnkeyData, which has no canonical status because a
//     "data" error can be a 404, 409, 410, and so on. Every data code that
//     surfaces over HTTP needs an entry.
//  2. Its category default is wrong for this specific code, for example an
//     "application" code (default 500) that is really a 400 or a 412.
//
// A code absent from this map resolves via Category.HTTPStatus(). Keep the list
// small: if many codes in a category need the same override, the category is
// probably modelling the wrong thing. The map keys on Code directly (Code is
// comparable), so a parsed URN resolves to the same status as the declared
// constant without any registry or wire-level Kind.
var codeStatusOverrides = map[Code]HTTPStatus{
	// CategoryUnkeyData: no category default, resolved per code.
	Data.Key.NotFound:                StatusNotFound,
	Data.Workspace.NotFound:          StatusNotFound,
	Data.Api.NotFound:                StatusNotFound,
	Data.Migration.NotFound:          StatusNotFound,
	Data.KeySpace.NotFound:           StatusNotFound,
	Data.Project.NotFound:            StatusNotFound,
	Data.Permission.NotFound:         StatusNotFound,
	Data.Permission.Duplicate:        StatusConflict,
	Data.Role.NotFound:               StatusNotFound,
	Data.Role.Duplicate:              StatusConflict,
	Data.KeyAuth.NotFound:            StatusNotFound,
	Data.RatelimitNamespace.NotFound: StatusNotFound,
	Data.RatelimitNamespace.Gone:     StatusGone,
	Data.RatelimitOverride.NotFound:  StatusNotFound,
	Data.Identity.NotFound:           StatusNotFound,
	Data.Identity.Duplicate:          StatusConflict,
	Data.AuditLog.NotFound:           StatusNotFound,
	Data.PortalConfig.NotFound:       StatusNotFound,
	Data.Analytics.NotConfigured:     StatusPreconditionFailed,
	Data.Analytics.ConnectionFailed:  StatusServiceUnavailable,

	// CategoryUnkeyApplication defaults to 500; these are client-facing.
	App.Validation.InvalidInput:         StatusBadRequest,
	App.Protection.ProtectedResource:    StatusPreconditionFailed,
	App.Precondition.PreconditionFailed: StatusPreconditionFailed,

	// CategoryUserBadRequest defaults to 400; these need a more specific status.
	User.BadRequest.RequestBodyTooLarge: StatusRequestEntityTooLarge,
	User.BadRequest.RequestTimeout:      StatusRequestTimeout,
	User.BadRequest.ClientClosedRequest: StatusClientClosedRequest,
}

// HTTPStatus returns the HTTP status a client should see for this code.
// Resolution order:
//
//  1. A per-code override (codeStatusOverrides), for codes whose status is not
//     implied by their category.
//  2. The Category default (Category.HTTPStatus()).
//  3. Zero, meaning the code has no canonical HTTP mapping. Callers at an HTTP
//     boundary should treat zero as a bug (the code was emitted from a non-HTTP
//     context) and respond 500.
func (c Code) HTTPStatus() HTTPStatus {
	if s, ok := codeStatusOverrides[c]; ok {
		return s
	}

	return c.Category.HTTPStatus()
}
