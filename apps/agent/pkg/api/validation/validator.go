package validation

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
)

type OpenAPIValidator interface {
	Body(r *http.Request, bodyPointer any) error
}

type Validator struct {
	validator validator.Validator
}

func New(specPath string) (*Validator, error) {
	b, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to read spec file"))
	}
	document, err := libopenapi.NewDocument(b)
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
func (v *Validator) Body(r *http.Request, bodyPointer any) error {

	valid, errors := v.validator.ValidateHttpRequestSync(r)
	if !valid {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fmsg.With(e.Error())
		}
		return fault.New("request validation failed", messages...)
	}

	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err := dec.Decode(bodyPointer)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to parse request body"))
	}

	return nil

}
