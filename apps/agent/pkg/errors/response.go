package errors

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Util to quickly return an error
func NewHttpError(c *fiber.Ctx, code ErrorCode, message string) error {
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
	default:
		status = fiber.StatusInternalServerError
	}

	requestId, ok := c.Locals("requestId").(string)
	if ok {
		res.Error.RequestId = requestId
	}
	res.Error.Docs = fmt.Sprintf("https://docs.unkey.dev/errors/code/%s", code)
	return c.Status(status).JSON(res)
}
