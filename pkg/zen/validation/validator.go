package validation

import (
	"context"
	"net/http"
	"sync"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/config"
	validatorErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/helpers"
	"github.com/unkeyed/unkey/pkg/ctxutil"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/api/openapi"
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
}

func New() (*Validator, error) {
	document, err := libopenapi.NewDocument(openapi.Spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create OpenAPI document"))
	}

	v, errors := validator.NewValidator(document, config.WithRegexCache(&sync.Map{}))
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

	valid, errors := v.validator.ValidateHttpRequestSync(r)

	if !valid {
		errors = filterIgnoredSecurityErrors(errors)
		valid = len(errors) == 0
	}

	if valid {
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}
	res := openapi.BadRequestErrorResponse{
		Meta: openapi.Meta{
			RequestId: ctxutil.GetRequestID(r.Context()),
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
				Location: verr.KeywordLocation,
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

// filterIgnoredSecurityErrors drops OpenAPI security-scheme errors that our
// handlers already produce richer messages for. Specifically:
//
//   - "scheme mismatch" (added in libopenapi-validator v0.13): the handler's
//     bearer parser returns a more useful "missing 'Bearer ' prefix" error.
//
// A missing Authorization header is still surfaced by the validator so that
// the existing 400 invalid_input contract is preserved.
func filterIgnoredSecurityErrors(errs []*validatorErrors.ValidationError) []*validatorErrors.ValidationError {
	filtered := errs[:0]
	for _, e := range errs {
		if e.ValidationType == helpers.SecurityValidation && e.Reason == "Authorization header had incorrect scheme" {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}
