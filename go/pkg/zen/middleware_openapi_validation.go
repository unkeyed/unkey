package zen

import (
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

func WithValidation(validator *validation.Validator) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(s *Session) error {
			err, valid := validator.Validate(s.r)
			if !valid {
				err.RequestId = s.requestID
				return s.JSON(err.Status, err)
			}
			return next(s)
		}
	}
}
