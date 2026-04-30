package validation

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/ctxutil"
	core "github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Validator struct {
	core *core.Validator
}

func New() (*Validator, error) {
	v, err := core.NewFromBytes(openapi.Spec)
	if err != nil {
		return nil, err
	}
	return &Validator{core: v}, nil
}

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
