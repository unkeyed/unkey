package apierrors

type InternalServerError struct {
	BaseError
}

// InternalServerError creates a new error with defaults
//
// the request ID will be injected automatically
func NewInternalServerError(title, detail string) InternalServerError {
	return InternalServerError{
		BaseError{
			Type:      "https://unkey.com/docs/errors/internal_server_error",
			Status:    500,
			Title:     title,
			Detail:    detail,
			RequestID: "",
		},
	}

}
