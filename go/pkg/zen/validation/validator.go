package validation

import (
	"net/http"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/ctxutil"
	"github.com/unkeyed/unkey/go/pkg/fault"
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
		return nil, fault.Wrap(err, fault.WithDesc("failed to create OpenAPI document", ""))
	}

	v, errors := validator.NewValidator(document)
	if len(errors) > 0 {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fault.WithDesc(e.Error(), "")
		}
		// nolint:wrapcheck
		return nil, fault.New("failed to create validator", messages...)
	}
	valid, docErrors := v.ValidateDocument()
	if !valid {
		messages := make([]fault.Wrapper, len(docErrors))
		for i, e := range docErrors {
			messages[i] = fault.WithDesc(e.Message, "")
		}

		return nil, fault.New("openapi document is invalid", messages...)
	}
	return &Validator{
		validator: v,
	}, nil
}

func (v *Validator) Validate(r *http.Request) (openapi.BadRequestError, bool) {

	valid, errors := v.validator.ValidateHttpRequest(r)
	if valid {
		// nolint:exhaustruct
		return openapi.BadRequestError{}, true
	}
	valErr := openapi.BadRequestError{
		Title:     "Bad Request",
		Detail:    "One or more fields failed validation",
		Instance:  nil,
		Status:    http.StatusBadRequest,
		RequestId: ctxutil.GetRequestId(r.Context()),
		Type:      "https://unkey.com/docs/errors/bad_request",
		Errors:    []openapi.ValidationError{},
	}

	for _, err := range errors {

		for _, e := range err.SchemaValidationErrors {

			valErr.Errors = append(valErr.Errors, openapi.ValidationError{
				Message:  e.Reason,
				Location: e.AbsoluteLocation,
				Fix:      &err.HowToFix,
			})
		}

	}
	return valErr, false

}
