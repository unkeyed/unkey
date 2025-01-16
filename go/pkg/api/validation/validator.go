package validation

import (
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/go/pkg/api/ctxutil"
	"github.com/unkeyed/unkey/go/pkg/openapi"
)

type OpenAPIValidator interface {
	Query(r *http.Request, dest any) (openapi.ValidationError, bool)

	Body(r *http.Request, dest any) (openapi.ValidationError, bool)
}

type Validator struct {
	validator validator.Validator
}

func New() (*Validator, error) {

	document, err := libopenapi.NewDocument(openapi.Spec)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to create OpenAPI document"))
	}

	v, errors := validator.NewValidator(document)
	if len(errors) > 0 {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fmsg.With(e.Error())
		}
		return nil, fault.New("failed to create validator", messages...)
	}
	return &Validator{
		validator: v,
	}, nil
}

// Body reads the request body and validates it against the OpenAPI spec
// The body is closed after reading.
// Returns a ValidationError if the body is invalid that should be marshalled and returned to the client.
// The second return value is a boolean that is true if the body is valid.
func (v *Validator) Body(r *http.Request) (openapi.ValidationError, bool) {

	valid, errors := v.validator.ValidateHttpRequest(r)
	if !valid {
		valErr := openapi.ValidationError{
			Title:     "Bad Request",
			Detail:    "One or more fields failed validation",
			Instance:  "",
			Status:    http.StatusBadRequest,
			RequestId: ctxutil.GetRequestId(r.Context()),
			Type:      "https://unkey.com/docs/api-reference/errors/TODO",
			Errors:    []openapi.ValidationErrorDetail{},
		}
		if len(errors) >= 1 {
			error := errors[0]

			valErr.Title = error.Message
			valErr.Detail = error.HowToFix

			for _, e := range error.SchemaValidationErrors {

				valErr.Errors = append(valErr.Errors, openapi.ValidationErrorDetail{
					Message:  e.Reason,
					Location: e.AbsoluteLocation,
					Fix:      &error.HowToFix,
				})
			}

		}
		return valErr, false
	}

	return openapi.ValidationError{}, true

}

// Query reads the query params and validates it against the OpenAPI spec
//
// Returns a ValidationError if the query is invalid that should be marshalled and returned to the client.
// The second return value is a boolean that is true if the body is valid.
func (v *Validator) Query(r *http.Request, dest any) (openapi.ValidationError, bool) {

	valid, errors := v.validator.GetParameterValidator().ValidateQueryParams(r)
	if !valid {
		valErr := openapi.ValidationError{
			Title:     "Bad Request",
			Detail:    "One or more fields failed validation",
			Instance:  "",
			Status:    http.StatusBadRequest,
			RequestId: ctxutil.GetRequestId(r.Context()),
			Type:      "https://unkey.com/docs/api-reference/errors/TODO",
			Errors:    []openapi.ValidationErrorDetail{},
		}
		if len(errors) >= 1 {
			error := errors[0]

			valErr.Title = error.Message
			valErr.Detail = error.HowToFix

			for _, e := range error.SchemaValidationErrors {

				valErr.Errors = append(valErr.Errors, openapi.ValidationErrorDetail{
					Message:  e.Reason,
					Location: e.AbsoluteLocation,
					Fix:      &error.HowToFix,
				})
			}

		}
		return valErr, false
	}

	return openapi.ValidationError{}, true

}
