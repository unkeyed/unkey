---
title: zen
description: "provides a lightweight HTTP framework built on top of the standard library"
---

Package zen provides a lightweight HTTP framework built on top of the standard library.

Zen is designed to add minimal abstraction over Go's net/http package while providing convenient utilities for common web service patterns. It follows Go's philosophy of simplicity and explicitness, offering just enough structure to make HTTP handlers more maintainable without obscuring the underlying functionality.

Core concepts:

Session: A request/response context that simplifies parsing requests and sending responses. Sessions are pooled and reused to reduce memory allocations.

	Route: Represents an HTTP endpoint with its method, path, and handler. Routes can
	be decorated with middleware.

	Middleware: Functions that wrap handlers to provide cross-cutting functionality like
	logging, error handling, tracing, and validation.

	Server: Manages the HTTP server lifecycle and route registration.

Basic usage example:

	// Initialize a new server
	server, err := zen.New(zen.Config{
	    NodeID: "service-1",
	})
	if err != nil {
	    log.Fatalf("failed to create server: %v", err)
	}

	// Create a route with middleware
	route := zen.NewRoute("GET", "/users/:id", func(s *zen.Session) error {
	    id := s.Request().PathValue("id")
	    user, err := userService.FindByID(s.Context(), id)
	    if err != nil {
	        return err
	    }
	    return s.JSON(http.StatusOK, user)
	})

	// Register route with middleware
	server.RegisterRoute(
	    []zen.Middleware{
	        zen.WithTracing(),
	        zen.WithLogging(),
	        zen.WithErrorHandling(),
	    },
	    route,
	)

	// Create a listener and start the server
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
	    log.Fatalf("failed to create listener: %v", err)
	}
	err = server.Serve(ctx, listener)

Zen is optimized for building maintainable, observable web services with minimal external dependencies and strong integration with standard Go libraries.

## Constants

CATCHALL is a special method constant that indicates a route should handle all HTTP methods. When a route returns CATCHALL (empty string) from Method(), it will be registered without a method prefix, allowing it to match all HTTP methods.
```go
const CATCHALL = ""
```

```go
const (
	// DefaultRequestTimeout is the default timeout for API requests
	DefaultRequestTimeout = 30 * time.Second
)
```


## Variables

ErrHijackAfterError is returned when hijacking is attempted after an error was captured.
```go
var ErrHijackAfterError = errors.New("hijack not allowed after error captured")
```

ErrHijackNotSupported is returned when the underlying ResponseWriter does not support hijacking.
```go
var ErrHijackNotSupported = errors.New("hijack not supported")
```

ErrPushNotSupported is returned when the underlying ResponseWriter does not support HTTP/2 push.
```go
var ErrPushNotSupported = errors.New("push not supported")
```

```go
var redactionRules = []redactionRule{

	{
		regexp:      regexp.MustCompile(`"key"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"key": "[REDACTED]"`),
	},

	{
		regexp:      regexp.MustCompile(`"plaintext"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"plaintext": "[REDACTED]"`),
	},
}
```

sessionKey is the context key for storing the session pointer.
```go
var sessionKey = NewContextKey[*Session]("session")
```

```go
var skipHeaders = map[string]bool{
	"x-forwarded-proto": true,
	"x-forwarded-port":  true,
	"x-forwarded-for":   true,
	"x-amzn-trace-id":   true,
}
```


## Functions

### func Bearer

```go
func Bearer(s *Session) (string, error)
```

Bearer extracts and validates a Bearer token from the Authorization header. It returns the token string if present and properly formatted.

If the header is missing, malformed, or contains an empty token, an appropriate error is returned with the BAD\_REQUEST tag.

Example:

	token, err := zen.Bearer(sess)
	if err != nil {
	    return err
	}
	// Validate the token

### func BindBody

```go
func BindBody[T any](s *Session) (T, error)
```

BindBody binds the request body to the given struct. If it fails, an error is returned, that you can directly return from your handler.

### func WithSession

```go
func WithSession(ctx context.Context, session *Session) context.Context
```

WithSession stores a session pointer in the context, making it available to downstream handlers and packages for operations like adding headers.

This function enables patterns where middleware or handlers need to modify HTTP response headers from within cache operations or other utility functions that don't have direct access to the session. The session is stored using a private key type to prevent conflicts with other context values.

Parameters:

  - ctx: The parent context to extend with session storage
  - session: The zen session to store. Must not be nil.

Returns a new context with the session stored. The original context is not modified. The session can be retrieved later using SessionFromContext.

Usage example:

	ctx = zen.WithSession(ctx, session)
	// Now downstream code can access the session:
	if s, ok := zen.SessionFromContext(ctx); ok {
	    s.AddHeader("X-Custom", "value")
	}

This is commonly used in middleware that enables functionality like cache debug headers, where cache operations need to write response headers without requiring explicit session passing through all function calls.


## Types

### type Config

```go
type Config struct {

	// TLS configuration for HTTPS connections.
	// If this is provided, the server will use HTTPS.
	TLS *tls.Config

	Flags *Flags

	// EnableH2C enables HTTP/2 cleartext (h2c) support.
	// This allows HTTP/2 connections without TLS, useful for internal services.
	EnableH2C bool

	// MaxRequestBodySize sets the maximum allowed request body size in bytes.
	// If 0 or negative, no limit is enforced. Default is 0 (no limit).
	// This helps prevent DoS attacks from excessively large request bodies.
	MaxRequestBodySize int64

	// ReadTimeout is the maximum duration for reading the entire request, including the body.
	// If 0, defaults to 10 seconds.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response.
	// If 0, defaults to 20 seconds.
	// For proxy services, this should be longer than any downstream timeout.
	WriteTimeout time.Duration
}
```

Config configures the behavior of a Server instance.

### type ContextKey

```go
type ContextKey[T any] struct {
	name string
}
```

ContextKey provides type-safe context storage using generics. It eliminates the need for type assertions and provides compile-time type safety when storing and retrieving values from context.

Example usage:

	var userIDKey = zen.NewContextKey[string]("user_id")
	ctx = userIDKey.WithValue(ctx, "user123")
	userID, ok := userIDKey.FromContext(ctx)

#### func NewContextKey

```go
func NewContextKey[T any](name string) ContextKey[T]
```

NewContextKey creates a new typed context key with the given name. The name is used for debugging and doesn't need to be globally unique since the key itself is used as the context key.

#### func (ContextKey) FromContext

```go
func (k ContextKey[T]) FromContext(ctx context.Context) (T, bool)
```

FromContext retrieves a value from the context using this key. Returns the value and true if found, or the zero value and false if not found.

#### func (ContextKey) WithValue

```go
func (k ContextKey[T]) WithValue(ctx context.Context, value T) context.Context
```

WithValue stores a value in the context using this key. Returns a new context with the value stored.

### type ErrorCapturingWriter

```go
type ErrorCapturingWriter struct {
	http.ResponseWriter
	capturedError error
	headerWritten bool
}
```

ErrorCapturingWriter wraps a ResponseWriter to capture proxy errors without writing them to the client. This allows errors to be returned to the middleware for consistent error handling.

This is useful when using httputil.ReverseProxy where you want to handle proxy errors in your handler instead of letting the proxy write directly to the client.

#### func NewErrorCapturingWriter

```go
func NewErrorCapturingWriter(w http.ResponseWriter) *ErrorCapturingWriter
```

NewErrorCapturingWriter creates a new error capturing writer that wraps the given ResponseWriter.

#### func (ErrorCapturingWriter) Error

```go
func (w *ErrorCapturingWriter) Error() error
```

Error returns any error that was captured during proxy, or nil if no error occurred.

#### func (ErrorCapturingWriter) Flush

```go
func (w *ErrorCapturingWriter) Flush()
```

Flush implements http.Flusher for streaming responses. No-op when error captured (discarding response anyway). Ensures headers are written before flushing to support streaming.

#### func (ErrorCapturingWriter) Hijack

```go
func (w *ErrorCapturingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error)
```

Hijack implements http.Hijacker for WebSocket and connection takeover. Returns ErrHijackAfterError if an error was captured, as the connection state is undefined. Returns ErrHijackNotSupported if the underlying ResponseWriter doesn't support hijacking.

#### func (ErrorCapturingWriter) Push

```go
func (w *ErrorCapturingWriter) Push(target string, opts *http.PushOptions) error
```

Push implements http.Pusher for HTTP/2 server push. No-op returning ErrPushNotSupported when error captured or underlying writer doesn't support push.

#### func (ErrorCapturingWriter) SetError

```go
func (w *ErrorCapturingWriter) SetError(err error)
```

SetError captures an error. This is typically called by httputil.ReverseProxy's ErrorHandler.

#### func (ErrorCapturingWriter) Unwrap

```go
func (w *ErrorCapturingWriter) Unwrap() http.ResponseWriter
```

Unwrap returns underlying ResponseWriter for http.ResponseController.

#### func (ErrorCapturingWriter) Write

```go
func (w *ErrorCapturingWriter) Write(b []byte) (int, error)
```

Write implements http.ResponseWriter. If an error was captured, the body write is discarded to prevent partial responses from being sent to the client.

#### func (ErrorCapturingWriter) WriteHeader

```go
func (w *ErrorCapturingWriter) WriteHeader(statusCode int)
```

WriteHeader implements http.ResponseWriter. If an error was captured, the header write is discarded to prevent partial responses from being sent to the client.

### type EventBuffer

```go
type EventBuffer interface {
	BufferApiRequest(schema.ApiRequest)
}
```

### type Flags

```go
type Flags struct {
	// TestMode enables test mode, accepting certain headers from untrusted clients such as fake times for testing purposes.
	TestMode bool
}
```

Flags configures the behavior of a Server instance.

### type HandleFunc

```go
type HandleFunc func(ctx context.Context, sess *Session) error
```

HandleFunc is a function type that implements the Handler interface. It provides a convenient way to create handlers without defining new types.

### type Handler

```go
type Handler interface {
	// Handle processes an HTTP request encapsulated by the Session.
	// It should return an error if processing fails.
	Handle(ctx context.Context, sess *Session) error
}
```

Handler defines the interface for HTTP request handlers in the Zen framework. Implementations receive a Session and return an error if processing fails.

### type InstanceInfo

```go
type InstanceInfo struct {
	ID     string
	Region string
}
```

### type Middleware

```go
type Middleware func(handler HandleFunc) HandleFunc
```

Middleware transforms one handler into another, typically by adding behavior before and/or after the original handler executes.

Middleware is used to implement cross-cutting concerns like logging, authentication, error handling, and metrics collection.

#### func WithLogging

```go
func WithLogging() Middleware
```

WithLogging returns middleware that logs information about each request. It captures the method, path, status code, and processing time.

Example:

	server.RegisterRoute(
	    []zen.Middleware{zen.WithLogging()},
	    route,
	)

#### func WithMetrics

```go
func WithMetrics(eventBuffer EventBuffer, info InstanceInfo) Middleware
```

WithMetrics returns middleware that collects metrics about each request, including request counts, latencies, and status codes.

The metrics are buffered and periodically sent to an event buffer.

Example:

	server.RegisterRoute(
	    []zen.Middleware{zen.WithMetrics(eventBuffer)},
	    route,
	)

#### func WithObservability

```go
func WithObservability() Middleware
```

WithObservability returns middleware that adds OpenTelemetry metrics and tracing to each request. It creates a span for the entire request lifecycle and propagates context.

If an error occurs during handling, it will be recorded in the span.

Example:

	server.RegisterRoute(
	    []zen.Middleware{zen.WithObservability()},
	    route,
	)

#### func WithPanicRecovery

```go
func WithPanicRecovery() Middleware
```

WithPanicRecovery returns middleware that recovers from panics and converts them into appropriate HTTP error responses.

#### func WithTimeout

```go
func WithTimeout(timeout time.Duration) Middleware
```

WithTimeout returns middleware that enforces a timeout on request processing. It differentiates between client-initiated cancellations and server-side timeouts.

#### func WithValidation

```go
func WithValidation(validator *validation.Validator) Middleware
```

WithValidation returns middleware that validates incoming requests against an OpenAPI schema. Invalid requests receive a 400 Bad Request response with detailed validation errors.

Example:

	validator, err := validation.New()
	if err != nil {
	    log.Fatalf("failed to create validator: %v", err)
	}

	server.RegisterRoute(
	    []zen.Middleware{zen.WithValidation(validator)},
	    route,
	)

### type Route

```go
type Route interface {
	Handler

	// Method returns the HTTP method this route responds to (GET, POST, etc.).
	// Return CATCHALL to handle all HTTP methods.
	Method() string

	// Path returns the URL path pattern this route matches.
	Path() string
}
```

Route represents an HTTP endpoint with its method, path, and handler function. It encapsulates the behavior of a specific HTTP endpoint in the system.

### type Server

```go
type Server struct {
	mu sync.Mutex

	isListening bool
	mux         *http.ServeMux
	srv         *http.Server
	flags       Flags
	config      Config

	sessions sync.Pool
}
```

Server manages HTTP server configuration, route registration, and lifecycle. It provides connection pooling for session objects to reduce memory churn during request handling.

Server instances should be created with the New function and can be safely used by multiple goroutines.

#### func New

```go
func New(config Config) (*Server, error)
```

New creates a new server with the provided configuration. It initializes the HTTP server and session pool with default timeouts.

The HTTP server is configured with reasonable defaults for production use: - ReadTimeout: 10 seconds - WriteTimeout: 20 seconds

Example:

	server, err := zen.New(zen.Config{
	    InstanceID: "api-server-1",
	   ,
	})
	if err != nil {
	    log.Fatalf("failed to initialize server: %v", err)
	}

#### func (Server) Flags

```go
func (s *Server) Flags() Flags
```

#### func (Server) Mux

```go
func (s *Server) Mux() *http.ServeMux
```

Mux returns the underlying http.ServeMux. This is primarily intended for testing and advanced usage scenarios.

#### func (Server) RegisterRoute

```go
func (s *Server) RegisterRoute(middlewares []Middleware, route Route)
```

RegisterRoute adds an HTTP route to the server with the specified middleware chain. Routes are matched by both method and path, unless the method is CATCHALL (empty string) which matches all methods.

Middleware is applied in the order provided, with each middleware wrapping the next. The innermost handler (last to execute) is the route's handler.

Example:

	server.RegisterRoute(
	    []zen.Middleware{zen.WithLogging(), zen.WithErrorHandling()},
	    zen.NewRoute("GET", "/health", healthCheckHandler),
	)

	// Catch-all route that handles all methods
	server.RegisterRoute(
	    []zen.Middleware{zen.WithLogging()},
	    zen.NewRoute(zen.CATCHALL, "/{path...}", proxyHandler),
	)

#### func (Server) Serve

```go
func (s *Server) Serve(ctx context.Context, ln net.Listener) error
```

Listen starts the HTTP server on the specified address. This method blocks until the server shuts down or encounters an error. Once listening, the server will not start again if Listen is called multiple times. If TLS configuration is provided, the server will use HTTPS.

The provided context is used to gracefully shut down the server when the context is canceled.

Example:

	// Start server in a goroutine to allow for graceful shutdown
	go func() {
	    if err := server.Listen(ctx, ":8080"); err != nil {
	        log.Printf("server stopped: %v", err)
	    }
	}()

#### func (Server) Shutdown

```go
func (s *Server) Shutdown(ctx context.Context) error
```

Shutdown gracefully stops the HTTP server, allowing in-flight requests to complete before returning or the context is canceled.

Example:

	// Handle shutdown signal
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
	    log.Printf("server shutdown error: %v", err)
	}

### type Session

```go
type Session struct {
	requestID string

	w http.ResponseWriter // Wrapped with statusRecorder to capture status code
	r *http.Request

	// The workspace making the request.
	// We extract this from the root key or regular key
	// and must set it before the metrics middleware finishes.
	WorkspaceID string

	requestBody    []byte
	responseStatus int
	responseBody   []byte

	// ClickHouse request logging control - defaults to true (log by default)
	logRequestToClickHouse bool

	// internalError stores the internal error message for logging to ClickHouse.
	// This is set by the error handling middleware before it converts the error
	// to an HTTP response, allowing the metrics middleware to log the full error.
	internalError string
}
```

Session encapsulates the state and utilities for handling a single HTTP request. It wraps the standard http.ResponseWriter and http.Request with additional functionality for parsing requests and generating responses.

Sessions are pooled and reused between requests to reduce memory allocations. References to sessions, requests, or responses should not be stored beyond the handler's execution.

A new Session is created for each request and passed to the route handler. The Session is automatically reset and returned to the pool after the request is handled.

#### func SessionFromContext

```go
func SessionFromContext(ctx context.Context) (*Session, bool)
```

SessionFromContext retrieves the session pointer stored by WithSession.

This function allows utility packages and handlers to access the HTTP session for operations like adding response headers, reading request data, or accessing session metadata. The session is safely type-cast from the context value.

Parameters:

  - ctx: The context to search for a stored session

Returns:

  - session: The stored session pointer, or nil if no session was found
  - ok: true if a session was found, false otherwise

The boolean return follows Go conventions for optional values and allows callers to distinguish between "no session stored" and "session stored but nil". However, WithSession should never store a nil session in practice.

Usage example:

	session, ok := zen.SessionFromContext(ctx)
	if !ok {
	    // No session available - cache debug disabled
	    return
	}
	session.AddHeader("X-Cache-Debug", "api_by_id:150μs:FRESH")

Performance note: Context value lookup is O(depth) where depth is the number of nested context.WithValue calls. This is typically very fast (\<100ns) for normal request contexts, but avoid calling this in tight loops.

#### func (Session) AddHeader

```go
func (s *Session) AddHeader(key, val string)
```

AddHeader adds a key-value pair to the response headers. This method can be called multiple times with the same key to add multiple values for the same header.

#### func (Session) AuthorizedWorkspaceID

```go
func (s *Session) AuthorizedWorkspaceID() string
```

AuthorizedWorkspaceID returns the workspace ID associated with the authenticated request. This is populated by authentication middleware.

Returns an empty string if no authenticated workspace ID is available.

#### func (Session) BindBody

```go
func (s *Session) BindBody(dst any) error
```

BindBody parses the request body as JSON into the provided destination struct. The destination must be a pointer to a struct.

If parsing fails, an appropriate error is returned. The original request body is stored in the session for potential reuse or logging.

Example:

	var user User
	if err := sess.BindBody(&user); err != nil {
	    return err
	}
	// Use the parsed user data

#### func (Session) BindQuery

```go
func (s *Session) BindQuery(dst interface{}) error
```

BindQuery parses URL query parameters into the provided destination struct. The destination must be a pointer to a struct with json tags that match the query parameter names.

Example:

	var params struct {
	    Limit  int    `json:"limit"`
	    Cursor string `json:"cursor"`
	    Filter string `json:"filter"`
	}
	if err := sess.BindQuery(&params); err != nil {
	    return err
	}
	// Use params.Limit, params.Cursor, and params.Filter

#### func (Session) DisableClickHouseLogging

```go
func (s *Session) DisableClickHouseLogging()
```

DisableClickHouseLogging prevents this request from being logged to ClickHouse. By default, all requests are logged to ClickHouse unless explicitly disabled.

This is useful for internal endpoints like health checks, OpenAPI specs, or requests that should not appear in analytics.

#### func (Session) HTML

```go
func (s *Session) HTML(status int, body []byte) error
```

HTML sends an HTML response with the given status code.

#### func (Session) Init

```go
func (s *Session) Init(w http.ResponseWriter, r *http.Request, maxBodySize int64) error
```

#### func (Session) InternalError

```go
func (s *Session) InternalError() string
```

InternalError returns the stored internal error message for logging.

#### func (Session) JSON

```go
func (s *Session) JSON(status int, body any) error
```

JSON sets the response status code and sends a JSON-encoded response. It automatically sets the Content-Type header to application/json.

The body is marshaled using github.com/bytedance/sonic If marshaling fails, an error is returned.

Example:

	return sess.JSON(http.StatusOK, map[string]interface{}{
	    "user": user,
	    "token": token,
	})

#### func (Session) Location

```go
func (s *Session) Location() string
```

Location returns the client's IP address, checking X-Forwarded-For header first, then falling back to RemoteAddr. Ports are stripped from the returned IP.

#### func (Session) Plain

```go
func (s *Session) Plain(status int, body []byte) error
```

Plain sends a plain text response with the given status code.

#### func (Session) Request

```go
func (s *Session) Request() *http.Request
```

Request returns the underlying http.Request. This allows direct access to the standard library request features.

Note: The returned request should not be stored across requests or modified after the handler returns.

#### func (Session) RequestID

```go
func (s *Session) RequestID() string
```

RequestID returns the request id for this session.

#### func (Session) ResponseWriter

```go
func (s *Session) ResponseWriter() http.ResponseWriter
```

ResponseWriter returns the http.ResponseWriter with status code capturing. This allows direct access to the standard library response features.

Direct manipulation of the ResponseWriter should be avoided when possible in favor of using the Session's response methods like JSON or Send.

#### func (Session) Send

```go
func (s *Session) Send(status int, body []byte) error
```

Send sets the response status code and sends raw bytes as the response body. This method is useful for non-JSON responses like binary data or plain text.

Unlike \[JSON], this method does not set any Content-Type header automatically.

#### func (Session) SetInternalError

```go
func (s *Session) SetInternalError(err string)
```

SetInternalError stores the internal error message for logging purposes. This should be called by error handling middleware before converting errors to HTTP responses.

#### func (Session) ShouldLogRequestToClickHouse

```go
func (s *Session) ShouldLogRequestToClickHouse() bool
```

ShouldLogRequestToClickHouse returns whether this request should be logged to ClickHouse. Returns true by default, false only if explicitly disabled.

#### func (Session) StatusCode

```go
func (s *Session) StatusCode() int
```

StatusCode returns the HTTP status code that was written to the response. Returns 200 if no status code has been explicitly set.

#### func (Session) UserAgent

```go
func (s *Session) UserAgent() string
```

