package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

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

// Util to quickly return an error
func newHttpError(c *fiber.Ctx, code ErrorCode, message string) error {
	res := ErrorResponse{}
	res.Error.Code = code
	res.Error.Message = message

	var status int
	switch code {
	case NOT_FOUND:
		status = fiber.StatusNotFound
	case BAD_REQUEST:
		status = fiber.StatusBadRequest
	case UNAUTHORIZED:
		status = fiber.StatusUnauthorized
	case FORBIDDEN:
		status = fiber.StatusForbidden
	case KEY_USAGE_EXCEEDED, RATELIMITED:
		status = fiber.StatusOK
	case TODO, INTERNAL_SERVER_ERROR:
		status = fiber.StatusInternalServerError
	case NOT_UNIQUE:
		status = fiber.StatusConflict
	default:
		status = fiber.StatusInternalServerError
	}

	requestId, ok := c.Locals("requestId").(string)
	if ok {
		res.Error.RequestId = requestId
	}
	res.Error.Docs = fmt.Sprintf("https://unkey.dev/docs/api-reference/errors/code/%s", code)
	return c.Status(status).JSON(res)
}

func fromServiceError(c *fiber.Ctx, err error) error {
	var svcErr errors.ServiceError
	if errors.As(err, &svcErr) {
		switch svcErr.Code {
		case errors.ErrNotFound:
			return newHttpError(c, NOT_FOUND, svcErr.Message.Error())
		case errors.ErrBadRequest:
			return newHttpError(c, BAD_REQUEST, svcErr.Message.Error())
		case errors.ErrUnauthorized:
			return newHttpError(c, UNAUTHORIZED, svcErr.Message.Error())
		case errors.ErrForbidden:
			return newHttpError(c, FORBIDDEN, svcErr.Message.Error())
		case errors.ErrKeyUsageExceeded:
			return newHttpError(c, KEY_USAGE_EXCEEDED, svcErr.Message.Error())
		case errors.ErrRatelimited:
			return newHttpError(c, RATELIMITED, svcErr.Message.Error())
		case errors.ErrNotUnique:
			return newHttpError(c, NOT_UNIQUE, svcErr.Message.Error())
		default:
			return newHttpError(c, INTERNAL_SERVER_ERROR, svcErr.Message.Error())
		}
	}

	return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
}
