package validation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/unkeyed/unkey/go/apps/gw/server"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

// Service implements the Validator interface with caching support.
type Service struct {
	logger           logging.Logger
	openapiSpecCache cache.Cache[string, validator.Validator]
}

// New creates a new validation service.
func New(config Config) (*Service, error) {
	if err := assert.All(
		assert.NotNilAndNotZero(config.Logger, "Logger is required"),
		assert.NotNilAndNotZero(config.OpenAPISpecCache, "OpenAPI spec cache is required"),
	); err != nil {
		return nil, err
	}

	return &Service{
		logger:           config.Logger,
		openapiSpecCache: config.OpenAPISpecCache,
	}, nil
}

// Validate implements the Validator interface.
func (s *Service) Validate(ctx context.Context, sess *server.Session, config *partitionv1.GatewayConfig) error {
	// Skip validation if not enabled
	if config.ValidationConfig == nil || !config.ValidationConfig.Enabled {
		return nil
	}

	// Skip if no OpenAPI spec is configured
	if config.ValidationConfig.OpenapiSpec == "" {
		s.logger.Warn("validation enabled but no OpenAPI spec configured",
			"requestId", sess.RequestID(),
			"deploymentId", config.DeploymentId,
		)
		return nil
	}

	// Start tracing span
	ctx, span := tracing.Start(ctx, "gateway.validation.Validate")
	defer span.End()

	// Get or create validator for this spec
	v, err := s.getOrCreateValidator(ctx, config.DeploymentId, config.ValidationConfig.OpenapiSpec)
	if err != nil {
		s.logger.Error("failed to get validator",
			"requestId", sess.RequestID(),
			"deploymentId", config.DeploymentId,
			"error", err.Error(),
		)
		// Don't fail the request if we can't create a validator
		// This allows the gateway to continue functioning even if validation setup fails
		return nil
	}

	// Perform validation
	req := sess.Request()
	valid, errors := (*v).ValidateHttpRequest(req)

	if valid {
		s.logger.Debug("request validation passed",
			"requestId", sess.RequestID(),
			"deploymentId", config.DeploymentId,
			"method", req.Method,
			"path", req.URL.Path,
		)
		return nil
	}

	// Build validation error response
	validationErr := s.buildValidationError(errors)

	s.logger.Warn("request validation failed",
		"requestId", sess.RequestID(),
		"deploymentId", config.DeploymentId,
		"method", req.Method,
		"path", req.URL.Path,
		"errors", len(validationErr.Errors),
	)

	// Return error wrapped with proper fault codes
	return fault.Wrap(validationErr,
		fault.Code(codes.Gateway.Validation.RequestInvalid.URN()),
		fault.Public(validationErr.Detail),
	)
}

// getOrCreateValidator retrieves a cached validator or creates a new one.
func (s *Service) getOrCreateValidator(ctx context.Context, deploymentID string, spec string) (*validator.Validator, error) {
	ctx, span := tracing.Start(ctx, "gateway.validation.getOrCreateValidator")
	defer span.End()

	// Create cache key based on deployment ID (spec content could be large)
	cacheKey := fmt.Sprintf("validator:%s", deploymentID)

	// Try to get from cache
	cachedValidator, cacheResult := s.openapiSpecCache.Get(ctx, cacheKey)
	if cacheResult == cache.Hit {
		s.logger.Debug("validator cache hit",
			"deploymentId", deploymentID,
		)
		return &cachedValidator, nil
	}

	// Cache miss - create new validator
	s.logger.Debug("validator cache miss, creating new validator",
		"deploymentId", deploymentID,
	)

	// Parse OpenAPI document
	document, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Internal("failed to parse OpenAPI document"),
		)
	}

	// Create validator
	v, validationErrors := validator.NewValidator(document)
	if len(validationErrors) > 0 {
		messages := make([]fault.Wrapper, len(validationErrors))
		for i, e := range validationErrors {
			messages[i] = fault.Internal(e.Error())
		}
		return nil, fault.New("failed to create validator", messages...)
	}

	// Validate the document itself
	valid, docErrors := v.ValidateDocument()
	if !valid {
		messages := make([]fault.Wrapper, len(docErrors))
		for i, e := range docErrors {
			messages[i] = fault.Internal(e.Message)
		}
		return nil, fault.New("OpenAPI document is invalid", messages...)
	}

	// Store in cache
	s.openapiSpecCache.Set(ctx, cacheKey, v)

	s.logger.Info("created and cached new validator",
		"deploymentId", deploymentID,
	)

	return &v, nil
}

// buildValidationError converts OpenAPI validation errors to our error format.
func (s *Service) buildValidationError(validationErrors []*errors.ValidationError) ValidationError {
	ve := ValidationError{
		StatusCode: http.StatusBadRequest,
		Title:      "Request Validation Failed",
		Detail:     "The request does not conform to the API specification",
		Errors:     make([]FieldError, 0),
	}

	if len(validationErrors) == 0 {
		return ve
	}

	// Use the first error's message as the main detail
	firstErr := validationErrors[0]
	ve.Detail = firstErr.Message

	// Convert all errors to field errors
	for _, err := range validationErrors {
		// Handle schema validation errors
		if len(err.SchemaValidationErrors) > 0 {
			for _, schemaErr := range err.SchemaValidationErrors {
				fe := FieldError{
					Field:   schemaErr.Location,
					Message: schemaErr.Reason,
				}
				ve.Errors = append(ve.Errors, fe)
			}
		} else {
			// Handle general validation error
			fe := FieldError{
				Field:   err.ValidationType,
				Message: err.Reason,
			}
			if err.HowToFix != "" {
				fe.Fix = &err.HowToFix
			}
			ve.Errors = append(ve.Errors, fe)
		}
	}

	return ve
}
