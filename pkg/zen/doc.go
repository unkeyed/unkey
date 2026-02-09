// Package zen provides a lightweight HTTP framework built on top of the standard library.
//
// Zen is designed to add minimal abstraction over Go's net/http package while providing
// convenient utilities for common web service patterns. It follows Go's philosophy of
// simplicity and explicitness, offering just enough structure to make HTTP handlers
// more maintainable without obscuring the underlying functionality.
//
// Core concepts:
//
// Session: A request/response context that simplifies parsing requests and sending
// responses. Sessions are pooled and reused to reduce memory allocations.
//
//	Route: Represents an HTTP endpoint with its method, path, and handler. Routes can
//	be decorated with middleware.
//
//	Middleware: Functions that wrap handlers to provide cross-cutting functionality like
//	logging, error handling, tracing, and validation.
//
//	Server: Manages the HTTP server lifecycle and route registration.
//
// Basic usage example:
//
//	// Initialize a new server
//	server, err := zen.New(zen.Config{
//	    NodeID: "service-1",
//	})
//	if err != nil {
//	    log.Fatalf("failed to create server: %v", err)
//	}
//
//	// Create a route with middleware
//	route := zen.NewRoute("GET", "/users/:id", func(s *zen.Session) error {
//	    id := s.Request().PathValue("id")
//	    user, err := userService.FindByID(s.Context(), id)
//	    if err != nil {
//	        return err
//	    }
//	    return s.JSON(http.StatusOK, user)
//	})
//
//	// Register route with middleware
//	server.RegisterRoute(
//	    []zen.Middleware{
//	        zen.WithTracing(),
//	        zen.WithLogging(),
//	        zen.WithErrorHandling(),
//	    },
//	    route,
//	)
//
//	// Create a listener and start the server
//	listener, err := net.Listen("tcp", ":8080")
//	if err != nil {
//	    log.Fatalf("failed to create listener: %v", err)
//	}
//	err = server.Serve(ctx, listener)
//
// Zen is optimized for building maintainable, observable web services with minimal
// external dependencies and strong integration with standard Go libraries.
package zen
