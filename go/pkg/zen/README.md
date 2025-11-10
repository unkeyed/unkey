<div align="center">
    <h1 align="center">Zen</h1>
    <p>A Minimalist HTTP Library for Go</p>
    <p><a href="http://www.unkey.com/blog/zen">Read our blog post</a> about why we built Zen and how it works</p>
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
- Support for HTTPS connections with TLS certificates.
- Context-based graceful shutdown handling.
- No bloat—just the essentials.
- Fast and easy to integrate with existing Go projects.

## Quickstart

```go
package main

import (
	"context"
	"log"
	"log/slog"
	"net"
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
		req, err := zen.BindBody[CreateUserRequest](s)
		if err != nil {
			return err
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
			zen.WithObservability(),
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

	// Create a listener
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to create listener: %v", err)
	}

	err = server.Serve(context.Background(), listener)
	if err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
	}
}
```

## Using TLS for HTTPS

To start the server with HTTPS, simply provide TLS certificate and key data:

```go
package main

import (
	"context"
	"log"
	"net"

	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func main() {
	// Load TLS configuration from certificate and key files
	tlsConfig, err := tls.NewFromFiles("server.crt", "server.key")
	if err != nil {
		log.Fatalf("failed to load TLS configuration: %v", err)
	}

	// Create a server with TLS configuration
	server, err := zen.New(zen.Config{
		TLS: tlsConfig,
	})
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Register routes...

	// Start the HTTPS server with context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a listener for HTTPS
	listener, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Fatalf("failed to create listener: %v", err)
	}

	// Start in a goroutine so you can handle shutdown signals
	go func() {
		if err := server.Serve(ctx, listener); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	// ...

	// To shut down gracefully:
	cancel() // This will initiate graceful shutdown
}
```

## Testing with Ephemeral Ports

For testing, you can use ephemeral ports to let the OS assign an available port automatically. This prevents port conflicts in testing environments:

```go
import "github.com/unkeyed/unkey/go/pkg/listener"

// Get an available port and listener
listenerImpl, err := listener.Ephemeral()
if err != nil {
	t.Fatalf("failed to create ephemeral listener: %v", err)
}
netListener, err := listenerImpl.Listen()
if err != nil {
	t.Fatalf("failed to get listener: %v", err)
}

// Start the server
go server.Serve(ctx, netListener)

// Make requests to the server
resp, err := http.Get(fmt.Sprintf("http://%s/test", listenerImpl.Addr()))
```

This approach is especially useful for concurrent tests where multiple servers need to run simultaneously without conflicting ports.

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

## Security

Zen supports HTTPS connections with TLS configuration. We recommend using TLS in production environments to encrypt all client-server communication. The TLS implementation uses Go's standard library crypto/tls package with secure defaults (TLS 1.2+). For TLS configuration management, Zen uses the unkey/go/pkg/tls package which provides utilities for creating TLS configurations from certificates and keys.

## Graceful Shutdown

Zen provides built-in support for graceful shutdown through context cancellation:

```go
// Create a context that can be cancelled
ctx, cancel := context.WithCancel(context.Background())

// Create a listener and start the server with this context
listener, err := net.Listen("tcp", ":8080")
if err != nil {
	log.Fatalf("failed to create listener: %v", err)
}

go server.Serve(ctx, listener)

// When you need to shut down (e.g., on SIGTERM):
cancel()

// For more control over the shutdown timeout:
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer shutdownCancel()
err := server.Shutdown(shutdownCtx)
```

When a server's context is canceled, it will:

1. Stop accepting new connections
2. Complete any in-flight requests
3. Release resources and exit gracefully
