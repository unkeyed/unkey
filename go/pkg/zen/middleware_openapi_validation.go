package zen

import (
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

func WithValidation(validator *validation.Validator) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(s *Session) error {
			fmt.Println("Middleware WithValidation")
			err, valid := validator.Validate(s.r)
			if !valid {
				err.RequestId = s.requestID
				return s.JSON(err.Status, err)
			}
			return next(s)
		}
	}
}
