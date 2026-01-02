package zen

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen/validation"
)

// WithValidation returns middleware that validates incoming requests against
// an OpenAPI schema. Invalid requests receive a 400 Bad Request response with
// detailed validation errors.
//
// Example:
//
//	validator, err := validation.New()
//	if err != nil {
//	    log.Fatalf("failed to create validator: %v", err)
//	}
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithValidation(validator)},
//	    route,
//	)
func WithValidation(validator *validation.Validator) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			err, valid := validator.Validate(ctx, s.r)
			if !valid {
				err.Meta.RequestId = s.requestID
				return s.JSON(err.Error.Status, err)
			}

			return next(ctx, s)
		}
	}
}
