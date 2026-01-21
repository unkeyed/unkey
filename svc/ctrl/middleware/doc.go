// Package middleware provides authentication and authorization for control plane APIs.
//
// This package implements simple API key authentication middleware for
// protecting Connect endpoints. It validates Bearer tokens against
// expected API key and provides error responses for failed authentication.
//
// # Authentication Flow
//
// 1. Extract Authorization header from incoming request
// 2. Validate Bearer token scheme format
// 3. Extract API key from token
// 4. Compare against expected API key
// 5. Allow or deny request based on validation
//
// The middleware skips authentication for health check endpoints
// to enable monitoring and load balancer health checks.
//
// # Key Types
//
// [AuthMiddleware]: Main middleware implementation
// [AuthConfig]: Configuration for API key validation
//
// # Usage
//
// Setting up authentication middleware:
//
//	auth := middleware.NewAuthMiddleware(middleware.AuthConfig{
//		APIKey: "expected-api-key-here",
//	})
//
//	// Apply to connect handler
//	handler := auth.ConnectInterceptor()(originalHandler)
//
// # Errors
//
// The middleware provides standardized error responses:
//   - ErrMissingAuthHeader: Authorization header missing
//   - ErrInvalidAuthScheme: Not using Bearer scheme
//   - ErrInvalidAPIKey: API key doesn't match expected value
//
// These errors are wrapped with appropriate Connect error codes
// for Connect transmission.
package middleware
