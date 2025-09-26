package middleware

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

// AuthConfig contains configuration for the authentication middleware
type AuthConfig struct {
	// APIKey is the expected API key for authentication
	APIKey string
}

// AuthMiddleware provides simple API key authentication
// TODO: Replace with JWT authentication when moving to private IP
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
			// TODO: Replace with JWT token extraction when moving to private IP
			authHeader := strings.TrimSpace(req.Header().Get("Authorization"))
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("Missing Authorization header"))
			}

			// Parse authorization header with case-insensitive Bearer scheme
			const bearerScheme = "bearer"
			if len(authHeader) < len(bearerScheme)+1 {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("Invalid Authorization header format. Expected: Bearer <api_key>"))
			}

			// Extract scheme and check case-insensitively
			schemePart := strings.ToLower(authHeader[:len(bearerScheme)])
			if schemePart != bearerScheme {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("Invalid Authorization header format. Expected: Bearer <api_key>"))
			}

			// Ensure there's a space after the scheme
			if authHeader[len(bearerScheme)] != ' ' {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("Invalid Authorization header format. Expected: Bearer <api_key>"))
			}

			// Extract and trim the token
			apiKey := strings.TrimSpace(authHeader[len(bearerScheme)+1:])
			if apiKey == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("API key cannot be empty"))
			}

			// Simple API key validation against environment variable
			// TODO: Replace with JWT validation when moving to private IP
			if apiKey != m.config.APIKey {
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("Invalid API key"))
			}

			// Continue to next handler
			return next(ctx, req)
		}
	}
}