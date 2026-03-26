package validation

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/config"
	validatorerrs "github.com/pb33f/libopenapi-validator/errors"
	"github.com/unkeyed/unkey/pkg/ctxutil"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen/metrics"
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

	// libopenapi-validator can produce spurious "circular reference detected
	// during inline rendering" errors under concurrent access. The upstream
	// library added InlineRenderContext (pb33f/libopenapi#488) to isolate
	// per-call cycle tracking, but the rendering pipeline still shares
	// mutable document state (lazy SchemaProxy initialization, NodeBuilder
	// reflection reads) that isn't fully synchronized. This manifests as
	// a transient false-positive from SchemaProxy.marshalYAMLInlineInternal.
	//
	// Retry up to 3 times on that specific error — a subsequent attempt
	// succeeds once the conflicting goroutine's lazy init has settled.
	//
	// We're planning to move away from libopenapi-validator, so this
	// quickfix is acceptable rather than investing in a proper fix.
	var valid bool
	var errors []*validatorerrs.ValidationError
	for range 1 {
		valid, errors = v.validator.ValidateHttpRequest(r)
		if valid || !hasCircularRefError(errors) {
			break
		}
		metrics.OpenAPIValidationRetryTotal.Inc()
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

func hasCircularRefError(errors []*validatorerrs.ValidationError) bool {
	for _, e := range errors {
		if strings.Contains(e.Message, "circular reference detected") {
			return true
		}
	}
	return false
}
