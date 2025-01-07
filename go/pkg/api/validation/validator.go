package validation

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/ctxutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/go/pkg/api/openapi"
)

type OpenAPIValidator interface {
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
func (v *Validator) Body(body []byte, dest any) (openapi.ValidationError, bool) {
	v.validator.ValidateHttpRequest(request * http.Request)
	valid, errors := v.validator.ValidateHttpRequest(body)
	if !valid {
		valErr := openapi.ValidationError{
			Title:     "Bad Request",
			Detail:    "One or more fields failed validation",
			Instance:  "https://errors.unkey.com/todo",
			Status:    http.StatusBadRequest,
			RequestId: ctxutil.GetRequestId(r.Context()),
			Type:      "TODO docs link",
			Errors:    []openapi.ValidationErrorDetail{},
		}
		for _, e := range errors {
			fmt.Printf("valerr: %+v\n", e)

			for _, schemaValidationError := range e.SchemaValidationErrors {
				valErr.Errors = append(valErr.Errors, openapi.ValidationErrorDetail{
					Location: schemaValidationError.Location,
					Message:  schemaValidationError.Reason,
				})
			}
		}
		return valErr, false
	}

	err = json.Unmarshal(bodyBytes, dest)

	if err != nil {
		return openapi.ValidationError{
			Title:  "Bad Request",
			Detail: "Failed to parse request body as JSON",
			Errors: []openapi.ValidationErrorDetail{{
				Location: "body",
				Message:  err.Error(),
				Fix:      util.Pointer("Ensure the request body is valid JSON"),
			}},
			Instance:  "https://errors.unkey.com/todo",
			Status:    http.StatusBadRequest,
			RequestId: ctxutil.GetRequestId(r.Context()),
			Type:      "TODO docs link",
		}, false
	}
	fmt.Printf("body %+v\n", dest)

	return openapi.ValidationError{}, true

}
