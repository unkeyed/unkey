package codes

import (
	"fmt"
	"net/http"
)

// HTTPStatus is a typed HTTP status code. Values must be in 100–599.
// Construct via NewHTTPStatus or one of the StatusXxx constants below;
// raw int conversion is discouraged outside this package.
//
// Existing as a dedicated type (rather than reusing int) means function
// signatures like `func httpStatus(urn) HTTPStatus` communicate intent
// at the call site, the value can't be confused with other ints
// (timeouts, ports, byte counts), and validation lives in one place.
type HTTPStatus int

// Status constants for the HTTP responses we emit. The list is the union
// of statuses currently used across our error-response middlewares.
// Pulled from net/http to preserve the standard library as the source of
// truth for the underlying numbers; 499 ("client closed request") is
// non-stdlib but widely used.
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

// NewHTTPStatus validates and constructs an HTTPStatus.
// Panics if status is outside 100–599. Codes are package-level vars,
// so an invalid status crashes the service at init, never at request time.
func NewHTTPStatus(status int) HTTPStatus {
	if status < 100 || status > 599 {
		panic(fmt.Sprintf("codes: invalid HTTP status %d", status))
	}
	return HTTPStatus(status)
}

// Int returns the underlying integer for use with http.ResponseWriter.WriteHeader.
func (s HTTPStatus) Int() int { return int(s) }

// Text returns the canonical reason phrase for s ("Not Found", "Unauthorized", …).
// Returns the empty string for non-standard codes that net/http does not recognise.
func (s HTTPStatus) Text() string { return http.StatusText(int(s)) }

// HTTPStatus returns the canonical HTTP status for this category.
// Returns 0 for categories that are not surfaced as HTTP responses
// (e.g. internal data errors, vault errors). A zero return signals
// to callers that the URN should not have leaked to an HTTP boundary.
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
		// Data is too broad to have a single canonical HTTP status;
		// individual codes carry a Kind (KindNotFound, KindDuplicate,
		// KindGone, …) which drives the response.
		return 0
	case CategoryUnkeyVault:
		// Vault is internal infrastructure; leaks to HTTP are bugs.
		return 0
	}
	return 0
}

// HTTPStatus returns the HTTP status that callers should use when
// reporting this code to a client. Resolution order:
//
//  1. Code-level Kind via Kind.HTTPStatus(). Use this for cross-cutting
//     semantics (e.g. KindNotFound, KindDuplicate) that may not match
//     the category default.
//  2. Category default via Category.HTTPStatus().
//  3. Zero, meaning the code has no canonical HTTP mapping. Callers
//     that surface errors over HTTP should treat zero as a bug
//     (the code was emitted from a non-HTTP context) and respond 500.
func (c Code) HTTPStatus() HTTPStatus {
	if s := c.Kind.HTTPStatus(); s != 0 {
		return s
	}
	return c.Category.HTTPStatus()
}
