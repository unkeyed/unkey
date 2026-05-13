package validation

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/ctxutil"
	core "github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Validator wraps the core OpenAPI validator with tracing and
// API-specific error formatting.
type Validator struct {
	core *core.Validator
}

// New creates a Validator backed by the compiled API OpenAPI spec.
func New() (*Validator, error) {
	v, err := core.NewFromBytes(openapi.Spec)
	if err != nil {
		return nil, err
	}
	return &Validator{core: v}, nil
}

// Validate checks r against the OpenAPI spec.
// Returns (_, true) when valid. On failure returns a ready-to-marshal
// BadRequestErrorResponse and false.
func (v *Validator) Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool) {
	_, span := tracing.Start(ctx, "openapi.Validate")
	defer span.End()

	result := v.core.Validate(r)
	if result == nil {
		//nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	errors := make([]openapi.ValidationError, len(result.Errors))
	for i, e := range result.Errors {
		errors[i] = openapi.ValidationError{
			Message:  e.Message,
			Location: e.Location,
			Fix:      e.Fix,
		}
	}

	//nolint:exhaustruct
	return openapi.BadRequestErrorResponse{
		Meta: openapi.Meta{
			RequestId: ctxutil.GetRequestID(r.Context()),
		},
		Error: openapi.BadRequestErrorDetails{
			Title:  "Bad Request",
			Detail: result.Detail,
			Status: http.StatusBadRequest,
			Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
			Errors: errors,
		},
	}, false
}
