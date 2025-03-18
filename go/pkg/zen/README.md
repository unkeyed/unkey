<div align="center">
    <h1 align="center">Zen</h1>
    <p>A Minimalist HTTP Library for Go</p>
    <p><a href="http://localhost:3000/blog/zen">Read our blog post</a> about why we built Zen and how it works</p>
</div>


Zen is a lightweight, minimalistic HTTP framework for Go, designed to wrap the
standard library with just enough abstraction to streamline your development
process—nothing more, nothing less.

## Why "Zen"?
The name "Zen" reflects the philosophy behind the framework: simplicity,
clarity, and efficiency.

- Simplicity: Focus on what matters most—handling HTTP requests and responses
  with minimal overhead.
- Clarity: A thin wrapper that feels natural, staying true to Go's idiomatic
  style.
- Efficiency: Built on Go's robust standard library, Zen adds no unnecessary
  complexity or dependencies.

## Features
- Built directly on the Go standard library (net/http).
- Thin abstractions for routing, middleware, and error handling.
- No bloat—just the essentials.
- Fast and easy to integrate with existing Go projects.

## Quickstart

```go
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Request struct for our create user endpoint
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Response for successful user creation
type CreateUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Initialize logger
	logger := logging.New()

	// Create a new server
	server, err := zen.New(zen.Config{
		NodeID: "quickstart-server",
		Logger: logger,
	})
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Initialize OpenAPI validator
	// see the validation package how we pass in the openapi spec
	validator, err := validation.New()
	if err != nil {
		log.Fatalf("failed to create validator: %v", err)
	}

	// Simple hello world route
	helloRoute := zen.NewRoute("GET", "/hello", func(ctx context.Context, s *zen.Session) error {
		return s.JSON(http.StatusOK, map[string]string{
			"message": "Hello, world!",
		})
	})

	// POST endpoint with request validation and error handling
	createUserRoute := zen.NewRoute("POST", "/users", func(ctx context.Context, s *zen.Session) error {
		// Parse request body
		var req CreateUserRequest
		if err := s.BindBody(&req); err != nil {
			return err // This will be handled by error middleware
		}

		// Additional validation logic
		if len(req.Password) < 8 {
			return fault.New("password too short",
				fault.WithTag(fault.BAD_REQUEST),
				fault.WithDesc(
					"password must be at least 8 characters", // Internal description
					"Password must be at least 8 characters long" // User-facing message
				),
			)
		}

		// Process the request (in a real app, you'd save to database etc.)
		userID := "user_pretendthisisrandom"

		// Return response
		return s.JSON(http.StatusCreated, CreateUserResponse{
			ID:    userID,
			Name:  req.Name,
			Email: req.Email,
		})
	})

	// Register routes with middleware
	server.RegisterRoute(
		[]zen.Middleware{
			zen.WithLogging(logger),
			zen.WithErrorHandling(logger),
		},
		helloRoute,
	)

	server.RegisterRoute(
		[]zen.Middleware{
			zen.WithTracing(),
			zen.WithLogging(logger),
			zen.WithErrorHandling(logger),
			zen.WithValidation(validator),
		},
		createUserRoute,
	)

	// Start the server
	logger.Info("starting server",
		"address", ":8080",
	)
	err = server.Listen(context.Background(), ":8080")
	if err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
	}
}
```

## Working with OpenAPI Validation

Zen works well with a schema-first approach to API design. Define your OpenAPI specification first, then use it for validation:

1. Define your OpenAPI spec
2. Initialize the validator with your spec
3. Add the validation middleware to your routes
4. Return properly tagged errors from your handlers

This approach ensures your API implementation strictly follows your API contract and provides excellent error messages to clients.

## Philosophy

Zen is for developers who embrace Go's simplicity and power. By focusing only
on essential abstractions, it keeps your code clean, maintainable, and in
harmony with Go's design principles.
