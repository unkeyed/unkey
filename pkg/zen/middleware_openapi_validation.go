package zen

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/svc/api/openapi"
)

// RequestValidator validates an HTTP request against the API contract.
type RequestValidator interface {
	Validate(ctx context.Context, req *http.Request) (openapi.BadRequestErrorResponse, bool)
}

// WithValidation returns middleware that validates incoming requests against
// an OpenAPI schema. Invalid requests receive a 400 Bad Request response with
// detailed validation errors.
func WithValidation(validator RequestValidator) Middleware {
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
