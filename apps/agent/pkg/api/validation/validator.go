package validation

import (
	"net/http"
	"os"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/gofiber/fiber/v2"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type OpenAPIValidator interface {
	Body(c *fiber.Ctx, bodyPointer any) error
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

func (v *Validator) Body(c *fiber.Ctx, bodyPointer any) error {
	httpReq := &http.Request{}
	err := fasthttpadaptor.ConvertRequest(c.Context(), httpReq, false)
	if err != nil {
		return fault.Wrap(err, fmsg.WithDesc("cannot convert request", "cannot convert request to http.Request"))
	}

	valid, errors := v.validator.ValidateHttpRequestSync(httpReq)
	if !valid {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fmsg.With(e.Error())
		}
		return fault.New("request validation failed", messages...)
	}

	err = c.BodyParser(bodyPointer)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to parse request body"))
	}

	return nil

}
