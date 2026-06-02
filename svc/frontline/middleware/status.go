package middleware

import (
	"github.com/unkeyed/unkey/pkg/codes"
)

// frontlineStatusOverrides holds the cases where frontline emits a different
// HTTP status than the code's canonical mapping in pkg/codes.
//
// Add an entry only when frontline's role as a gateway changes the semantics of
// an error. Most codes resolve fine via Code.HTTPStatus().
//
//nolint:exhaustive // sparse override map; not every URN has an override.
var frontlineStatusOverrides = map[codes.URN]codes.HTTPStatus{
	// The canonical RequestTimeout maps to 408 (the request itself took too
	// long). In frontline that almost always means our upstream proxy hop did
	// not return in time, which from the caller's perspective is a 504 Gateway
	// Timeout, not a 408. Surface it that way so the status reflects what
	// actually happened and does not trip request-error alerting.
	codes.User.BadRequest.RequestTimeout.URN(): codes.StatusGatewayTimeout,
}

// frontlineTitleOverrides holds per-URN page titles that differ from the
// standard reason phrase (HTTPStatus.Text()).
//
//nolint:exhaustive // sparse override map; not every URN has an override.
var frontlineTitleOverrides = map[codes.URN]string{
	codes.User.BadRequest.ClientClosedRequest.URN(): "Client Closed Request", // 499 has no stdlib name
}

// frontlineErrorMessages holds the user-facing copy rendered on the error page
// (or JSON body) per URN. A URN absent from this map falls back to the error's
// own UserFacingMessage. Status and title are NOT here: status comes from
// httpStatus(urn) and title from errorTitle(status, urn). This map only carries
// product-specific phrasing.
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

// httpStatus resolves the HTTP status frontline should return for urn.
// Resolution order:
//
//  1. Per-service override (frontlineStatusOverrides).
//  2. Per-code mapping in pkg/codes (Code.HTTPStatus()).
//  3. 500, when the URN is malformed or the code lives in a non-HTTP category
//     and leaked into a request path. Both are bugs worth alerting on.
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

// errorTitle returns the page title for the response. It defaults to the
// standard reason phrase (HTTPStatus.Text()); special cases live in
// frontlineTitleOverrides.
func errorTitle(status codes.HTTPStatus, urn codes.URN) string {
	if t, ok := frontlineTitleOverrides[urn]; ok {
		return t
	}

	return status.Text()
}
