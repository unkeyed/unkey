package middleware

import (
	"github.com/unkeyed/unkey/pkg/codes"
)

// frontlineStatusOverrides holds the rare cases where frontline emits a
// different HTTP status than the code's canonical mapping in pkg/codes.
//
// Add an entry only when frontline's role as a gateway changes the
// semantics of an error. Most codes resolve fine via Code.HTTPStatus().
var frontlineStatusOverrides = map[codes.URN]codes.HTTPStatus{
	// Canonical RequestTimeout maps to 408 (the request itself took too
	// long). In frontline that almost always means our upstream proxy
	// hop did not return in time, which from the caller's perspective
	// is a 504 Gateway Timeout, not a 408. Surface it that way so the
	// status reflects what actually happened.
	codes.User.BadRequest.RequestTimeout.URN(): codes.StatusGatewayTimeout,
}

// httpStatus resolves the HTTP status frontline should return for urn.
// Resolution order:
//
//  1. Per-service override (frontlineStatusOverrides).
//  2. Per-code mapping in pkg/codes (Code.HTTPStatus()).
//  3. 500 — either the URN is malformed or the code lives in a
//     non-HTTP category and leaked into a request path. Both are bugs
//     worth alerting on.
func httpStatus(urn codes.URN) codes.HTTPStatus {
	if s, ok := frontlineStatusOverrides[urn]; ok {
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
