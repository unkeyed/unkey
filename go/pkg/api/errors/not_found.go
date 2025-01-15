package apierrors

import (
	"context"
)

type NotFoundError struct {
	BaseError
}

// NewNotFoundError creates a new error with defaults and extracts the
// requestID from context
func NewNotFoundError(ctx context.Context, title, detail string) NotFoundError {
	requestID, ok := ctx.Value("request_id").(string)
	if !ok {
		requestID = ""
	}
	return NotFoundError{
		BaseError{
			Type:      "https://unkey.com/docs/errors/not_found",
			Status:    404,
			Title:     title,
			Detail:    detail,
			RequestID: requestID,
		},
	}

}
