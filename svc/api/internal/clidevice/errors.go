package clidevice

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func invalid(message string) error {
	return fault.New(message,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public(message),
	)
}

func forbidden(message string) error {
	return fault.New(message,
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Public(message),
	)
}

func unauthenticated(message string) error {
	return fault.New(message,
		fault.Code(codes.Auth.Authentication.Malformed.URN()),
		fault.Public(message),
	)
}

func loginNotFound() error {
	return invalid("This login code was not found. Run unkey login again.")
}

func alreadyIssued() error {
	return invalid("This login already issued a CLI session. If this CLI did not store it, revoke that session and run unkey login again.")
}
