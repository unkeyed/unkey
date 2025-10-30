package validation

import (
	"context"
	"net/http"
	"sync"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/ctxutil"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

type OpenAPIValidator interface {
	// Validate reads the request and validates it against the OpenAPI spec
	//
	// Returns a BadRequestError if the request is invalid that should be
	// marshalled and returned to the client.
	// The second return value is a boolean that is true if the request is valid.
	Validate(r *http.Request) (openapi.BadRequestErrorResponse, bool)
}

type Validator struct {
	validator validator.Validator
	// mu protects concurrent access to the validator to work around race conditions
	// in the underlying libopenapi-validator library (v0.9.0 and earlier)
	mu sync.Mutex
}

func New() (*Validator, error) {
	document, err := libopenapi.NewDocument(openapi.Spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create OpenAPI document"))
	}

	v, errors := validator.NewValidator(document)
	if len(errors) > 0 {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fault.Internal(e.Error())
		}
		// nolint:wrapcheck
		return nil, fault.New("failed to create validator", messages...)
	}
	valid, docErrors := v.ValidateDocument()
	if !valid {
		messages := make([]fault.Wrapper, len(docErrors))
		for i, e := range docErrors {
			messages[i] = fault.Internal(e.Message)
		}

		return nil, fault.New("openapi document is invalid", messages...)
	}

	return &Validator{
		validator: v,
	}, nil
}

func (v *Validator) Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool) {
	_, validationSpan := tracing.Start(ctx, "openapi.Validate")
	defer validationSpan.End()

	// Lock to protect concurrent access to the validator
	// This works around race conditions in libopenapi-validator v0.9.0 and earlier
	v.mu.Lock()
	valid, errors := v.validator.ValidateHttpRequest(r)
	v.mu.Unlock()

	if valid {
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}
	res := openapi.BadRequestErrorResponse{
		Meta: openapi.Meta{
			RequestId: ctxutil.GetRequestId(r.Context()),
		},
		Error: openapi.BadRequestErrorDetails{
			Title:  "Bad Request",
			Detail: "One or more fields failed validation",
			Status: http.StatusBadRequest,
			Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
			Errors: []openapi.ValidationError{},
		},
	}

	if len(errors) > 0 {
		err := errors[0]
		res.Error.Detail = err.Message
		for _, verr := range err.SchemaValidationErrors {
			res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
				Message:  verr.Reason,
				Location: verr.Location,
				Fix:      nil,
			})
		}
		if len(res.Error.Errors) == 0 {
			res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
				Message:  err.Reason,
				Location: err.ValidationType,
				Fix:      &err.HowToFix,
			})
		}
	}

	return res, false

}
