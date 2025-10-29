package middleware

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

var (
	ErrMissingAuthHeader = fmt.Errorf("missing Authorization header")
	ErrInvalidAuthScheme = fmt.Errorf("invalid Authorization scheme, expected 'Bearer <token>'")
	ErrInvalidAPIKey     = fmt.Errorf("invalid API key")
)

// AuthConfig contains configuration for the authentication middleware
type AuthConfig struct {
	// APIKey is the expected API key for authentication
	APIKey string
}

// AuthMiddleware provides simple API key authentication
type AuthMiddleware struct {
	config AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
	}
}

// ConnectInterceptor returns a Connect interceptor for gRPC-like services
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
