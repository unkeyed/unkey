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

// OpenAPIValidator defines the interface for validating HTTP requests against an OpenAPI spec
type OpenAPIValidator interface {
	// Validate reads the request and validates it against the OpenAPI spec
	//
	// Returns a BadRequestError if the request is invalid that should be
	// marshalled and returned to the client.
	// The second return value is a boolean that is true if the request is valid.
	Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool)
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

// Validate validates an HTTP request against the OpenAPI spec
func (v *Validator) Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool) {
	_, validationSpan := tracing.Start(ctx, "openapi.Validate")
	defer validationSpan.End()

	requestID := ctxutil.GetRequestId(ctx)

	// 1. Match request to operation
	matchResult, found := v.matcher.Match(r.Method, r.URL.Path)
	if !found {
		// No matching operation - pass through (let the router handle 404)
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	op := matchResult.Operation

	// 2. Validate security (fail fast)
	if err := v.validateSecurity(r, op.Security, requestID); err != nil {
		return *err, false
	}

	// 3. Get compiled operation schemas
	compiledOp := v.compiler.GetOperation(op.OperationID)

	// 4. Validate content-type for requests with body
	if r.ContentLength > 0 || r.Body != nil {
		if err := v.validateContentType(r, compiledOp, requestID); err != nil {
			return *err, false
		}
	}

	// 5. Validate parameters (aggregate errors)
	var paramErrors []openapi.ValidationError
	if compiledOp != nil {
		// Validate path parameters
		if len(compiledOp.Parameters.Path) > 0 && matchResult.PathParams != nil {
			paramErrors = append(paramErrors, v.validatePathParams(matchResult.PathParams, compiledOp.Parameters.Path)...)
		}
		if len(compiledOp.Parameters.Query) > 0 {
			paramErrors = append(paramErrors, v.validateQueryParams(r, compiledOp.Parameters.Query)...)
		}
		if len(compiledOp.Parameters.Header) > 0 {
			paramErrors = append(paramErrors, v.validateHeaderParams(r, compiledOp.Parameters.Header)...)
		}
		if len(compiledOp.Parameters.Cookie) > 0 {
			paramErrors = append(paramErrors, v.validateCookieParams(r, compiledOp.Parameters.Cookie)...)
		}
	}

	if len(paramErrors) > 0 {
		detail := "One or more parameters failed validation"
		if len(paramErrors) == 1 {
			detail = paramErrors[0].Message
		}
		return openapi.BadRequestErrorResponse{
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
		}, false
	}

	// 6. Validate body
	return v.validateBody(ctx, r, op, compiledOp)
}

// validateSecurity validates security requirements for the operation
func (v *Validator) validateSecurity(r *http.Request, requirements []SecurityRequirement, requestID string) *openapi.BadRequestErrorResponse {
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
