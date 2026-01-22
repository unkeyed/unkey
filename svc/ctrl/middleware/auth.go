package middleware

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

// Standard authentication errors returned by middleware.
var (
	// ErrMissingAuthHeader is returned when Authorization header is completely missing.
	ErrMissingAuthHeader = fmt.Errorf("missing Authorization header")

	// ErrInvalidAuthScheme is returned when Authorization header doesn't use Bearer scheme.
	ErrInvalidAuthScheme = fmt.Errorf("invalid Authorization scheme, expected 'Bearer <token>'")

	// ErrInvalidAPIKey is returned when API key doesn't match expected value.
	ErrInvalidAPIKey = fmt.Errorf("invalid API key")
)

// AuthConfig contains configuration for the authentication middleware.
//
// This configuration provides the expected API key that
// incoming requests must authenticate with using Bearer token.
type AuthConfig struct {
	// APIKey is the expected API key for authentication.
	// Requests must provide this value in Authorization header
	// to be allowed access to protected endpoints.
	APIKey string
}

// AuthMiddleware provides simple API key authentication.
//
// This middleware validates Bearer tokens against a configured API key.
// It allows unauthenticated access to health check endpoints
// but protects all other API endpoints.
type AuthMiddleware struct {
	// config holds the authentication configuration.
	config AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware.
//
// This function initializes an AuthMiddleware with the provided
// configuration containing the expected API key for validation.
//
// Returns a configured AuthMiddleware ready for use with Connect handlers.
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
	}
}

// ConnectInterceptor returns a Connect interceptor for Connect services.
//
// This method returns an interceptor function that validates Bearer tokens
// on incoming requests. It skips authentication for health check endpoints
// but protects all other API endpoints.
//
// The interceptor performs these validation steps:
// 1. Check for Authorization header presence
// 2. Validate Bearer scheme format (case-insensitive)
// 3. Extract and validate API key against expected value
// 4. Return appropriate error or continue to next handler
//
// Returns an interceptor function compatible with Connect middleware chains.
func (m *AuthMiddleware) ConnectInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Skip authentication for health check endpoint
			if procedure == "/ctrl.v1.CtrlService/Liveness" {
				return next(ctx, req)
			}

			// Extract API key from Authorization header
			authHeader := strings.TrimSpace(req.Header().Get("Authorization"))
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					ErrMissingAuthHeader)
			}

			// Parse authorization header with case-insensitive Bearer scheme
			const bearerScheme = "bearer"
			if len(authHeader) < len(bearerScheme)+1 {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					ErrInvalidAuthScheme)
			}

			// Extract scheme and check case-insensitively
			schemePart := strings.ToLower(authHeader[:len(bearerScheme)])
			if schemePart != bearerScheme {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					ErrInvalidAuthScheme)
			}

			// Ensure there's a space after the scheme
			if authHeader[len(bearerScheme)] != ' ' {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					ErrInvalidAuthScheme)
			}

			// Extract and trim the token
			apiKey := strings.TrimSpace(authHeader[len(bearerScheme)+1:])
			if apiKey == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("API key cannot be empty"))
			}

			// Simple API key validation against environment variable
			if apiKey != m.config.APIKey {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					ErrInvalidAPIKey)
			}

			// Continue to next handler
			return next(ctx, req)
		}
	}
}
