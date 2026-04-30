package validation

import (
	"net/http"
	"sync"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/config"
	validatorErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/helpers"
	"github.com/unkeyed/unkey/pkg/fault"
)

type ValidationError struct {
	Message  string
	Location string
	Fix      *string
}

type Result struct {
	Detail string
	Errors []ValidationError
}

type Validator struct {
	validator validator.Validator
}

func NewFromBytes(spec []byte) (*Validator, error) {
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create OpenAPI document"))
	}

	v, errors := validator.NewValidator(document, config.WithRegexCache(&sync.Map{}))
	if len(errors) > 0 {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fault.Internal(e.Error())
		}
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

	return &Validator{validator: v}, nil
}

func (v *Validator) Validate(r *http.Request) *Result {
	valid, errors := v.validator.ValidateHttpRequestSync(r)

	if !valid {
		errors = filterIgnoredSecurityErrors(errors)
		valid = len(errors) == 0
	}

	if valid {
		return nil
	}

	result := &Result{
		Detail: "One or more fields failed validation",
		Errors: []ValidationError{},
	}

	if len(errors) > 0 {
		err := errors[0]
		result.Detail = err.Message
		for _, verr := range err.SchemaValidationErrors {
			result.Errors = append(result.Errors, ValidationError{
				Message:  verr.Reason,
				Location: verr.KeywordLocation,
				Fix:      nil,
			})
		}
		if len(result.Errors) == 0 {
			howToFix := err.HowToFix
			result.Errors = append(result.Errors, ValidationError{
				Message:  err.Reason,
				Location: err.ValidationType,
				Fix:      &howToFix,
			})
		}
	}

	return result
}

func filterIgnoredSecurityErrors(errs []*validatorErrors.ValidationError) []*validatorErrors.ValidationError {
	filtered := make([]*validatorErrors.ValidationError, 0, len(errs))
	for _, e := range errs {
		if e.ValidationType == helpers.SecurityValidation && e.Reason == "Authorization header had incorrect scheme" {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}
