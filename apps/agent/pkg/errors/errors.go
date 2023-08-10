package errors

import (
	"errors"
)

// Reexported from "errors"
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Reexported from "errors"
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

type ErrorCode = string

const (
	TODO                  ErrorCode = "update this error code"
	NOT_FOUND             ErrorCode = "NOT_FOUND"
	BAD_REQUEST           ErrorCode = "BAD_REQUEST"
	UNAUTHORIZED          ErrorCode = "UNAUTHORIZED"
	INTERNAL_SERVER_ERROR ErrorCode = "INTERNAL_SERVER_ERROR"
	RATELIMITED           ErrorCode = "RATELIMITED"
	FORBIDDEN             ErrorCode = "FORBIDDEN"
	KEY_USAGE_EXCEEDED    ErrorCode = "KEY_USAGE_EXCEEDED"
	INVALID_KEY_TYPE      ErrorCode = "INVALID_KEY_TYPE"
)

// this is what a json body response looks like
type ErrorResponse struct {
	Error Error `json:"error"`
}

type Error struct {
	// A machine readable error code
	Code ErrorCode `json:"code"`

	// A link to our documentation explaining this error in more detail
	Docs string `json:"docs"`

	// A human readable short explanation
	Message string `json:"message"`

	// The request id for easy support lookup
	RequestId string `json:"requestId,omitempty"`
}
