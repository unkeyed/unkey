package apierrors

type NotFoundError struct {
	BaseError
}

// NewNotFoundError creates a new error with defaults
//
// the request ID will be injected automatically
func NewNotFoundError(title, detail string) NotFoundError {
	return NotFoundError{
		BaseError{
			Type:      "https://unkey.com/docs/errors/not_found",
			Status:    404,
			Title:     title,
			Detail:    detail,
			RequestID: "",
		},
	}

}
