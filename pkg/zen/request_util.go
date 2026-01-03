package zen

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// BindBody binds the request body to the given struct.
// If it fails, an error is returned, that you can directly return from your handler.
func BindBody[T any](s *Session) (T, error) {
	// nolint:exhaustruct
	var req T
	err := s.BindBody(&req)
	if err != nil {
		return req, fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("invalid request body"),
			fault.Public("The request body is invalid."),
		)
	}

	return req, nil
}
