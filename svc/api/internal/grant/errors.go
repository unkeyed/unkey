package grant

import (
	"errors"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ErrRejected marks a grant request or token its presenter cannot use: unknown,
// expired, denied, consumed, revoked, reused, or held by someone else. Errors
// that wrap it keep the fault code and public message the CLI routes render.
// OAuth routes match it with errors.Is and answer invalid_grant. An error that
// does not wrap it is an infrastructure failure.
var ErrRejected = errors.New("grant rejected")

// ErrUnknownClient means no registered OAuth client has the presented id.
var ErrUnknownClient = errors.New("unknown oauth client")

// ErrInvalidRedirectURI means a registration names a redirect URI the server
// does not accept. The public message names the rule it breaks.
var ErrInvalidRedirectURI = errors.New("invalid redirect uri")

// ErrInvalidClientMetadata means a registration's metadata other than its
// redirect URIs is unacceptable. The public message says what is wrong.
var ErrInvalidClientMetadata = errors.New("invalid client metadata")

func invalid(message string) error {
	return fault.New(message,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public(message),
	)
}

func rejectedInvalid(message string) error {
	return fault.Wrap(ErrRejected,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(message),
		fault.Public(message),
	)
}

func forbidden(message string) error {
	return fault.Wrap(ErrRejected,
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Internal(message),
		fault.Public(message),
	)
}

func unauthenticated(message string) error {
	return fault.Wrap(ErrRejected,
		fault.Code(codes.Auth.Authentication.CredentialRejected.URN()),
		fault.Internal(message),
		fault.Public(message),
	)
}

func loginNotFound() error {
	return rejectedInvalid("This login code was not found. Run unkey login again.")
}

func alreadyIssued() error {
	return rejectedInvalid("This login already issued a CLI session. If this CLI did not store it, revoke that session and run unkey login again.")
}

// rejectedCode is the one refusal every failed redemption gets, so the
// response does not reveal whether the code, client, redirect URI, or
// verifier was wrong.
func rejectedCode() error {
	return rejectedInvalid("The authorization code is invalid, expired, or already used.")
}
