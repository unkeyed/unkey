---
title: middleware
description: "provides authentication and authorization for control plane APIs"
---

Package middleware provides authentication and authorization for control plane APIs.

This package implements simple API key authentication middleware for protecting Connect endpoints. It validates Bearer tokens against expected API key and provides error responses for failed authentication.

### Authentication Flow

1\. Extract Authorization header from incoming request 2. Validate Bearer token scheme format 3. Extract API key from token 4. Compare against expected API key 5. Allow or deny request based on validation

The middleware skips authentication for health check endpoints to enable monitoring and load balancer health checks.

### Key Types

\[AuthMiddleware]: Main middleware implementation \[AuthConfig]: Configuration for API key validation

### Usage

Setting up authentication middleware:

	auth := middleware.NewAuthMiddleware(middleware.AuthConfig{
		APIKey: "expected-api-key-here",
	})

	// Apply to connect handler
	handler := auth.ConnectInterceptor()(originalHandler)

### Errors

The middleware provides standardized error responses:

  - ErrMissingAuthHeader: Authorization header missing
  - ErrInvalidAuthScheme: Not using Bearer scheme
  - ErrInvalidAPIKey: API key doesn't match expected value

These errors are wrapped with appropriate Connect error codes for Connect transmission.

## Variables

Standard authentication errors returned by middleware.
```go
var (
	// ErrMissingAuthHeader is returned when Authorization header is completely missing.
	ErrMissingAuthHeader = fmt.Errorf("missing Authorization header")

	// ErrInvalidAuthScheme is returned when Authorization header doesn't use Bearer scheme.
	ErrInvalidAuthScheme = fmt.Errorf("invalid Authorization scheme, expected 'Bearer <token>'")

	// ErrInvalidAPIKey is returned when API key doesn't match expected value.
	ErrInvalidAPIKey = fmt.Errorf("invalid API key")
)
```


## Types

### type AuthConfig

```go
type AuthConfig struct {
	// APIKey is the expected API key for authentication.
	// Requests must provide this value in Authorization header
	// to be allowed access to protected endpoints.
	APIKey string
}
```

AuthConfig contains configuration for the authentication middleware.

This configuration provides the expected API key that incoming requests must authenticate with using Bearer token.

### type AuthMiddleware

```go
type AuthMiddleware struct {
	// config holds the authentication configuration.
	config AuthConfig
}
```

AuthMiddleware provides simple API key authentication.

This middleware validates Bearer tokens against a configured API key. It allows unauthenticated access to health check endpoints but protects all other API endpoints.

#### func NewAuthMiddleware

```go
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware
```

NewAuthMiddleware creates a new authentication middleware.

This function initializes an AuthMiddleware with the provided configuration containing the expected API key for validation.

Returns a configured AuthMiddleware ready for use with Connect handlers.

#### func (AuthMiddleware) ConnectInterceptor

```go
func (m *AuthMiddleware) ConnectInterceptor() connect.UnaryInterceptorFunc
```

ConnectInterceptor returns a Connect interceptor for Connect services.

This method returns an interceptor function that validates Bearer tokens on incoming requests. It skips authentication for health check endpoints but protects all other API endpoints.

The interceptor performs these validation steps: 1. Check for Authorization header presence 2. Validate Bearer scheme format (case-insensitive) 3. Extract and validate API key against expected value 4. Return appropriate error or continue to next handler

Returns an interceptor function compatible with Connect middleware chains.

