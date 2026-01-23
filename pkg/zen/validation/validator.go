package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/unkeyed/unkey/pkg/ctxutil"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type OpenAPIValidator interface {
	// Validate reads the request and validates it against the OpenAPI spec
	//
	// Returns a BadRequestError if the request is invalid that should be
	// marshalled and returned to the client.
	// The second return value is a boolean that is true if the request is valid.
	Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool)
}

type Validator struct {
	matcher *PathMatcher
	schemas map[string]*jsonschema.Schema // operationID -> compiled schema
}

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

	// Build the schemas map from the compiler
	schemas := make(map[string]*jsonschema.Schema)
	for _, op := range parser.Operations() {
		if schema := compiler.Get(op.OperationID); schema != nil {
			schemas[op.OperationID] = schema
		}
	}

	return &Validator{
		matcher: matcher,
		schemas: schemas,
	}, nil
}

func (v *Validator) Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool) {
	_, validationSpan := tracing.Start(ctx, "openapi.Validate")
	defer validationSpan.End()

	// 1. Match request to operation
	op, found := v.matcher.Match(r.Method, r.URL.Path)
	if !found {
		// No matching operation - pass through (let the router handle 404)
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	// 2. Validate Authorization header if operation requires bearer auth
	if op.RequiresBearerAuth {
		if errResp, valid := validateAuthorizationHeader(r, ctxutil.GetRequestId(ctx)); !valid {
			return errResp, false
		}
	}

	// 3. Get the compiled schema for this operation
	schema, ok := v.schemas[op.OperationID]
	if !ok || schema == nil {
		// No request body schema for this operation - pass through
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	// 4. Read the request body
	if r.Body == nil {
		// No body provided but schema exists - validate empty
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: ctxutil.GetRequestId(ctx),
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Failed to read request body",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
				Errors: []openapi.ValidationError{},
			},
		}, false
	}

	// Reset body for downstream handlers
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Empty body - some operations may allow this
	if len(body) == 0 {
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	// 5. Unmarshal the body
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: ctxutil.GetRequestId(ctx),
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Invalid JSON in request body",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
				Errors: []openapi.ValidationError{
					{
						Location: "body",
						Message:  err.Error(),
						Fix:      ptr.P("Ensure the request body is valid JSON"),
					},
				},
			},
		}, false
	}

	// 6. Validate against the schema
	if err := schema.Validate(data); err != nil {
		return TransformErrors(err, ctxutil.GetRequestId(ctx)), false
	}

	// nolint:exhaustruct
	return openapi.BadRequestErrorResponse{}, true
}

// validateAuthorizationHeader checks that the Authorization header exists and uses Bearer scheme
func validateAuthorizationHeader(r *http.Request, requestID string) (openapi.BadRequestErrorResponse, bool) {
	authHeader := r.Header.Get("Authorization")

	// Check if Authorization header is present
	if authHeader == "" {
		return openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Authorization header is required but was not provided",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
				Errors: []openapi.ValidationError{
					{
						Location: "header.Authorization",
						Message:  "Authorization header is missing",
						Fix:      ptr.P("Add an Authorization header with format: Bearer <your-api-key>"),
					},
				},
			},
		}, false
	}

	// Check for Bearer scheme (case-insensitive)
	const bearerPrefix = "bearer "
	if len(authHeader) < len(bearerPrefix) || !strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
		return openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Authorization header must use Bearer scheme",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
				Errors: []openapi.ValidationError{
					{
						Location: "header.Authorization",
						Message:  "Authorization header must start with 'Bearer '",
						Fix:      ptr.P("Use format: Bearer <your-api-key>"),
					},
				},
			},
		}, false
	}

	// Check that there's actually a token after "Bearer "
	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	if token == "" {
		return openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Authorization header must use Bearer scheme",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
				Errors: []openapi.ValidationError{
					{
						Location: "header.Authorization",
						Message:  "Bearer token is empty",
						Fix:      ptr.P("Provide a valid API key after 'Bearer '"),
					},
				},
			},
		}, false
	}

	// nolint:exhaustruct
	return openapi.BadRequestErrorResponse{}, true
}
