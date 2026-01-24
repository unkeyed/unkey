package validation

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/ctxutil"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// ValidationErrorResponse is an interface for error responses returned by validation
type ValidationErrorResponse interface {
	GetStatus() int
	SetRequestID(requestID string)
}

// OpenAPIValidator defines the interface for validating HTTP requests against an OpenAPI spec
type OpenAPIValidator interface {
	// Validate reads the request and validates it against the OpenAPI spec
	//
	// Returns an error response if the request is invalid that should be
	// marshalled and returned to the client.
	// The second return value is a boolean that is true if the request is valid.
	Validate(ctx context.Context, r *http.Request) (ValidationErrorResponse, bool)
}

// Validator implements OpenAPIValidator using a parsed and compiled OpenAPI spec
type Validator struct {
	matcher         *PathMatcher
	compiler        *SchemaCompiler
	securitySchemes map[string]SecurityScheme
}

// New creates a new Validator from the embedded OpenAPI spec
func New() (*Validator, error) {
	parser, err := NewSpecParser(openapi.Spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to parse OpenAPI spec"))
	}

	compiler, err := NewSchemaCompiler(parser, openapi.Spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to compile schemas"))
	}

	matcher := NewPathMatcher(parser.Operations())

	return &Validator{
		matcher:         matcher,
		compiler:        compiler,
		securitySchemes: parser.SecuritySchemes(),
	}, nil
}

// BadRequestError wraps BadRequestErrorResponse to implement ValidationErrorResponse
type BadRequestError struct {
	openapi.BadRequestErrorResponse
}

func (e *BadRequestError) GetStatus() int {
	return e.Error.Status
}

func (e *BadRequestError) SetRequestID(requestID string) {
	e.Meta.RequestId = requestID
}

// UnauthorizedError wraps UnauthorizedErrorResponse to implement ValidationErrorResponse
type UnauthorizedError struct {
	openapi.UnauthorizedErrorResponse
}

func (e *UnauthorizedError) GetStatus() int {
	return e.Error.Status
}

func (e *UnauthorizedError) SetRequestID(requestID string) {
	e.Meta.RequestId = requestID
}

// Validate validates an HTTP request against the OpenAPI spec
func (v *Validator) Validate(ctx context.Context, r *http.Request) (ValidationErrorResponse, bool) {
	ctx, validationSpan := tracing.Start(ctx, "openapi.Validate")
	defer validationSpan.End()

	requestID := ctxutil.GetRequestId(ctx)

	// 1. Match request to operation
	matchResult, found := v.matcher.Match(r.Method, r.URL.Path)
	if !found {
		// No matching operation - pass through (let the router handle 404)
		return nil, true
	}

	op := matchResult.Operation

	// 2. Validate security (fail fast)
	_, secSpan := tracing.Start(ctx, "validation.ValidateSecurity")
	secErr := v.validateSecurity(r, op.Security, requestID)
	secSpan.End()
	if secErr != nil {
		return secErr, false
	}

	// 3. Get compiled operation schemas
	compiledOp := v.compiler.GetOperation(op.OperationID)

	// 4. Validate content-type for requests with body
	if r.ContentLength > 0 || r.Body != nil {
		_, ctSpan := tracing.Start(ctx, "validation.ValidateContentType")
		ctErr := v.validateContentType(r, compiledOp, requestID)
		ctSpan.End()
		if ctErr != nil {
			return ctErr, false
		}
	}

	// 5. Validate parameters (aggregate errors)
	var paramErrors []openapi.ValidationError
	if compiledOp != nil {
		// Validate path parameters
		if len(compiledOp.Parameters.Path) > 0 && matchResult.PathParams != nil {
			_, pathSpan := tracing.Start(ctx, "validation.ValidatePathParams")
			paramErrors = append(paramErrors, v.validatePathParams(matchResult.PathParams, compiledOp.Parameters.Path)...)
			pathSpan.End()
		}
		if len(compiledOp.Parameters.Query) > 0 {
			_, querySpan := tracing.Start(ctx, "validation.ValidateQueryParams")
			paramErrors = append(paramErrors, v.validateQueryParams(r, compiledOp.Parameters.Query)...)
			querySpan.End()
		}
		if len(compiledOp.Parameters.Header) > 0 {
			_, headerSpan := tracing.Start(ctx, "validation.ValidateHeaders")
			paramErrors = append(paramErrors, v.validateHeaderParams(r, compiledOp.Parameters.Header)...)
			headerSpan.End()
		}
		if len(compiledOp.Parameters.Cookie) > 0 {
			_, cookieSpan := tracing.Start(ctx, "validation.ValidateCookies")
			paramErrors = append(paramErrors, v.validateCookieParams(r, compiledOp.Parameters.Cookie)...)
			cookieSpan.End()
		}
	}

	if len(paramErrors) > 0 {
		detail := "One or more parameters failed validation"
		if len(paramErrors) == 1 {
			detail = paramErrors[0].Message
		}
		return &BadRequestError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: detail,
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
					Errors: paramErrors,
				},
			},
		}, false
	}

	// 6. Validate body
	_, bodySpan := tracing.Start(ctx, "validation.ValidateBody")
	result, valid := v.validateBody(ctx, r, compiledOp)
	bodySpan.End()
	return result, valid
}

// validateSecurity validates security requirements for the operation
func (v *Validator) validateSecurity(r *http.Request, requirements []SecurityRequirement, requestID string) *UnauthorizedError {
	// If no security requirements defined, the operation is public
	if len(requirements) == 0 {
		return nil
	}

	// For bearer auth, provide detailed error messages
	if v.requiresBearerAuth(requirements) {
		return ValidateBearerAuth(r, requestID)
	}

	// For other auth types, use generic validation
	return ValidateSecurity(r, requirements, v.securitySchemes, requestID)
}

// requiresBearerAuth checks if the security requirements include HTTP bearer auth
func (v *Validator) requiresBearerAuth(requirements []SecurityRequirement) bool {
	for _, req := range requirements {
		for schemeName := range req.Schemes {
			scheme, exists := v.securitySchemes[schemeName]
			if !exists {
				continue
			}
			if scheme.Type == SecurityTypeHTTP && strings.ToLower(scheme.Scheme) == "bearer" {
				return true
			}
		}
	}
	return false
}
