package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/ctxutil"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// validateContentType validates the Content-Type header against allowed content types
func (v *Validator) validateContentType(r *http.Request, compiledOp *CompiledOperation, requestID string) *BadRequestError {
	if compiledOp == nil || len(compiledOp.ContentTypes) == 0 {
		// No content type restrictions
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		// No Content-Type header provided - let body validation handle it
		return nil
	}

	// Parse the media type (ignoring parameters like charset)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return &BadRequestError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Invalid Content-Type header",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
					Errors: []openapi.ValidationError{
						{
							Location: "header.Content-Type",
							Message:  "Failed to parse Content-Type header: " + err.Error(),
							Fix:      ptr.P("Provide a valid Content-Type header (e.g., application/json)"),
						},
					},
					Schema: BuildSchemaURL(compiledOp.BodySchemaName),
				},
			},
		}
	}

	// Check if the media type is allowed
	for _, allowed := range compiledOp.ContentTypes {
		allowedMedia, _, _ := mime.ParseMediaType(allowed)
		if strings.EqualFold(mediaType, allowedMedia) {
			return nil
		}

		// Handle wildcards like application/*
		if strings.HasSuffix(allowedMedia, "/*") {
			prefix := strings.TrimSuffix(allowedMedia, "*")
			if strings.HasPrefix(mediaType, prefix) {
				return nil
			}
		}

		// Handle */* (accept all)
		if allowedMedia == "*/*" {
			return nil
		}
	}

	return &BadRequestError{
		BadRequestErrorResponse: openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Unsupported Content-Type: " + mediaType,
				Status: http.StatusUnsupportedMediaType,
				Type:   "https://unkey.com/docs/errors/unkey/application/unsupported_media_type",
				Errors: []openapi.ValidationError{
					{
						Location: "header.Content-Type",
						Message:  "Content-Type '" + mediaType + "' is not supported for this operation",
						Fix:      ptr.P("Use one of the supported content types: " + strings.Join(compiledOp.ContentTypes, ", ")),
					},
				},
				Schema: BuildSchemaURL(compiledOp.BodySchemaName),
			},
		},
	}
}

// validateBody validates the request body against the schema
func (v *Validator) validateBody(ctx context.Context, r *http.Request, compiledOp *CompiledOperation) (ValidationErrorResponse, bool) {
	requestID := ctxutil.GetRequestID(ctx)

	// Check if body is required
	if compiledOp != nil && compiledOp.BodyRequired {
		if r.Body == nil || r.ContentLength == 0 {
			// Try to read body to check if it's truly empty
			body, err := io.ReadAll(r.Body)
			if err == nil {
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
			if err != nil || len(body) == 0 {
				return &BadRequestError{
					BadRequestErrorResponse: openapi.BadRequestErrorResponse{
						Meta: openapi.Meta{
							RequestId: requestID,
						},
						Error: openapi.BadRequestErrorDetails{
							Title:  "Bad Request",
							Detail: "Request body is required but was not provided",
							Status: http.StatusBadRequest,
							Type:   "https://unkey.com/docs/errors/unkey/application/missing_body",
							Errors: []openapi.ValidationError{
								{
									Location: "body",
									Message:  "request body is required",
									Fix:      ptr.P("Provide a request body in the expected format"),
								},
							},
							Schema: BuildSchemaURL(compiledOp.BodySchemaName),
						},
					},
				}, false
			}
		}
	}

	// Check if we have a body schema to validate against
	if compiledOp == nil || compiledOp.BodySchema == nil {
		// No request body schema for this operation - pass through
		return nil, true
	}

	schema := compiledOp.BodySchema

	// Read the request body
	if r.Body == nil {
		// No body provided but schema exists - pass through
		return nil, true
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		schemaURL := BuildSchemaURL(compiledOp.BodySchemaName)
		return &BadRequestError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Failed to read request body",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
					Errors: []openapi.ValidationError{},
					Schema: schemaURL,
				},
			},
		}, false
	}

	// Reset body for downstream handlers
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Empty body - some operations may allow this
	if len(body) == 0 {
		return nil, true
	}

	// Unmarshal the body
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		schemaURL := BuildSchemaURL(compiledOp.BodySchemaName)
		return &BadRequestError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
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
					Schema: schemaURL,
				},
			},
		}, false
	}

	// Validate against the schema
	_, schemaSpan := tracing.Start(ctx, "validation.SchemaValidate")
	err = schema.Validate(data)
	schemaSpan.End()
	if err != nil {
		schemaName := ""
		if compiledOp != nil {
			schemaName = compiledOp.BodySchemaName
		}
		resp := TransformErrors(err, requestID, schemaName)
		return &BadRequestError{BadRequestErrorResponse: resp}, false
	}

	return nil, true
}
