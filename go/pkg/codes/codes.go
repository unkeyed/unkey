package codes

import (
	"fmt"
	"strings"
)

// Format constants define the structure of error code strings.
const (
	// Prefix is the standard prefix for all error codes.
	Prefix = "err"

	// Separator is used to separate parts of the error code.
	Separator = ":"
)

// System represents a service area in Unkey. Each error belongs to a specific
// system that indicates its origin or responsibility domain.
type System string

const (
	// SystemNil represents a nil or unknown error.
	SystemNil System = "nil"

	// SystemUser indicates errors caused by user inputs or client behavior.
	SystemUser System = "user"

	// SystemGateway indicates errors caused by gateway issues.
	SystemGateway System = "gateway"

	// SystemUnkey indicates errors originating from Unkey's internal systems.
	SystemUnkey System = "unkey"

	// SystemGitHub indicates errors related to GitHub integration.
	SystemGitHub System = "github"

	// SystemAws indicates errors related to AWS integration.
	SystemAws System = "aws"
)

// Category represents an error category within a system domain, providing
// a second level of classification for errors.
type Category string

const (
	// User categories

	// CategoryUserBadRequest represents invalid user input errors.
	CategoryUserBadRequest Category = "bad_request"

	// CategoryUserUnprocessableEntity represents requests that are syntactically correct but cannot be processed.
	CategoryUserUnprocessableEntity Category = "unprocessable_entity"

	// CategoryUserTooManyRequests represents rate limit exceeded errors.
	CategoryUserTooManyRequests Category = "too_many_requests"

	// CategoryNotFound represents resource not found errors.
	CategoryNotFound Category = "not_found"

	// CategoryBadGateway represents errors related to upstream server unavailability.
	CategoryBadGateway Category = "bad_gateway"

	// CategoryServiceUnavailable represents backend service unavailable errors.
	CategoryServiceUnavailable Category = "service_unavailable"

	// CategoryGatewayTimeout represents upstream server timeout errors.
	CategoryGatewayTimeout Category = "gateway_timeout"

	// CategoryUnauthorized represents authentication required or failed errors.
	CategoryUnauthorized Category = "unauthorized"

	// CategoryForbidden represents authorization/permission denied errors.
	CategoryForbidden Category = "forbidden"

	// CategoryRateLimited represents rate limit exceeded errors.
	CategoryRateLimited Category = "rate_limited"

	// CategoryInternalServerError represents internal server errors.
	CategoryInternalServerError Category = "internal_server_error"

	// Unkey categories

	// CategoryUnkeyData represents data-related errors in Unkey systems.
	CategoryUnkeyData Category = "data"

	// CategoryUnkeyAuthentication represents authentication failures.
	CategoryUnkeyAuthentication Category = "authentication"

	// CategoryUnkeyAuthorization represents authorization/permission failures.
	CategoryUnkeyAuthorization Category = "authorization"

	// CategoryUnkeyLimits represents rate limiting or quota-related errors.
	CategoryUnkeyLimits Category = "limits"

	CategoryUnkeyApplication Category = "application"

	// CategoryUnkeyVault represents vault-related errors.
	CategoryUnkeyVault Category = "vault"
)

// Code represents a specific error with its metadata. It contains all components
// needed to uniquely identify an error condition within Unkey.
type Code struct {
	// System identifies the service or domain this error belongs to.
	System System

	// Category identifies the error type within the system.
	Category Category

	// Specific identifies the exact error condition within the category.
	Specific string
}

// URN returns the URN-style string representation of the error code in the format
// "err:system:category:specific". This format is used for error serialization,
// logging, and cross-service communication.
func (c Code) URN() URN {
	return URN(strings.Join([]string{Prefix, string(c.System), string(c.Category), c.Specific}, Separator))
}

// DocsURL returns the documentation URL for this error code, providing users and
// developers with a direct link to detailed information about the error, including
// possible causes and remediation steps.
func (c Code) DocsURL() string {
	return fmt.Sprintf("https://unkey.com/docs/errors/%s/%s/%s",
		c.System, c.Category, c.Specific)
}

// ParseURN parses a URN string into a Code object. It provides a convenient way
// to convert error URNs back into structured Code objects. Returns an error if
// the URN format is invalid.
//
// See [ParseCode] for implementation details.
func ParseURN(urn URN) (Code, error) {
	return ParseCode(string(urn))
}

// ParseCode parses a URN-style error code string into a Code object. The expected
// format is "err:system:category:specific". Returns an error if the format is invalid.
func ParseCode(s string) (Code, error) {
	parts := strings.Split(s, Separator)
	if len(parts) != 4 || parts[0] != Prefix {
		return Code{}, fmt.Errorf("invalid error code format: %s", s)
	}

	return Code{
		System:   System(parts[1]),
		Category: Category(parts[2]),
		Specific: parts[3],
	}, nil
}
