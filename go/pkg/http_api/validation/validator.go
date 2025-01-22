package validation

import (
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/go/pkg/ctxutil"
	"github.com/unkeyed/unkey/go/pkg/http_api/openapi"
)

type OpenAPIValidator interface {

	// Validate reads the request and validates it against the OpenAPI spec
	//
	// Returns a BadRequestError if the request is invalid that should be
	// marshalled and returned to the client.
	// The second return value is a boolean that is true if the request is valid.
	Validate(r *http.Request) (openapi.BadRequestError, bool)
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
func (v *Validator) Validate(r *http.Request) (openapi.BadRequestError, bool) {

	valid, errors := v.validator.ValidateHttpRequest(r)
	if !valid {
		valErr := openapi.BadRequestError{
			Title:     "Bad Request",
			Detail:    "One or more fields failed validation",
			Instance:  nil,
			Status:    http.StatusBadRequest,
			RequestId: ctxutil.GetRequestId(r.Context()),
			Type:      "https://unkey.com/docs/api-reference/errors/TODO",
			Errors:    []openapi.ValidationError{},
		}
		if len(errors) >= 1 {
			error := errors[0]

			valErr.Title = error.Message
			valErr.Detail = error.HowToFix

			for _, e := range error.SchemaValidationErrors {

				valErr.Errors = append(valErr.Errors, openapi.ValidationError{
					Message:  e.Reason,
					Location: e.AbsoluteLocation,
					Fix:      &error.HowToFix,
				})
			}

		}
		return valErr, false
	}

	return openapi.BadRequestError{}, true

}
