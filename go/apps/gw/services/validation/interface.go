package validation

import (
	"context"

	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/go/apps/gw/server"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Validator defines the interface for OpenAPI request validation.
type Validator interface {
	// Validate checks the request against the OpenAPI specification.
	// Returns nil if validation succeeds or is not required.
	// Returns an error with validation details if the request is invalid.
	Validate(ctx context.Context, sess *server.Session, config *partitionv1.GatewayConfig) error
}

// Config holds configuration for the validator.
type Config struct {
	// Logger for debugging and monitoring
	Logger logging.Logger

	// Cache for storing OpenAPI validators
	OpenAPISpecCache cache.Cache[string, validator.Validator]
}

// ValidationError represents a validation failure
type ValidationError struct {
	StatusCode int
	Title      string
	Detail     string
	Errors     []FieldError
}

// FieldError represents a specific field validation error
type FieldError struct {
	Field   string
	Message string
	Fix     *string
}

func (e ValidationError) Error() string {
	return e.Detail
}

// ToHTTPResponse converts the validation error to an HTTP response format
func (e ValidationError) ToHTTPResponse() map[string]interface{} {
	errors := make([]map[string]interface{}, len(e.Errors))
	for i, err := range e.Errors {
		errors[i] = map[string]interface{}{
			"field":   err.Field,
			"message": err.Message,
		}
		if err.Fix != nil {
			errors[i]["fix"] = *err.Fix
		}
	}

	return map[string]interface{}{
		"error": map[string]interface{}{
			"title":  e.Title,
			"detail": e.Detail,
			"status": e.StatusCode,
			"errors": errors,
		},
	}
}
