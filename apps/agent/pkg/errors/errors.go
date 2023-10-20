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
	NOT_UNIQUE            ErrorCode = "NOT_UNIQUE"
)

var (
	ErrNotFound            = errors.New("NOT_FOUND")
	ErrBadRequest          = errors.New("BAD_REQUEST")
	ErrUnauthorized        = errors.New("UNAUTHORIZED")
	ErrInternalServerError = errors.New("INTERNAL_SERVER_ERROR")
	ErrRatelimited         = errors.New("RATELIMITED")
	ErrForbidden           = errors.New("FORBIDDEN")
	ErrKeyUsageExceeded    = errors.New("KEY_USAGE_EXCEEDED")
	ErrInvalidKeyType      = errors.New("INVALID_KEY_TYPE")
	ErrNotUnique           = errors.New("NOT_UNIQUE")
)

// this is what a json body response looks like
type ErrorResponse struct {
	Error ApplicationError `json:"error"`
}

type ApplicationError struct {
	// A machine readable error code
	Code ErrorCode `json:"code"`

	// A link to our documentation explaining this error in more detail
	Docs string `json:"docs"`

	// A human readable short explanation
	Message string `json:"message"`

	// The request id for easy support lookup
	RequestId string `json:"requestId,omitempty"`
}

func (e ApplicationError) Error() string {
	return e.Message
}

type Error struct {
	CodeError    error
	ServiceError error
}

func (e Error) Error() string {
	return errors.Join(e.CodeError, e.ServiceError).Error()
}

func New(code error, service error) error {
	return Error{
		CodeError:    code,
		ServiceError: service,
	}

}
