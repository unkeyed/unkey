---
title: proxy
description: "provides HTTP/HTTPS proxying services for the frontline"
---

Package proxy provides HTTP/HTTPS proxying services for the frontline.

The proxy service forwards requests to local sentinels or remote frontlines, manages a shared HTTP transport for connection pooling, writes timing headers for troubleshooting, and returns clean JSON or HTML error responses on failure.

### Header Management

The service writes identifying headers (frontline ID, region, request ID) on both responses and downstream requests. Timing details are recorded with the shared X-Unkey-Timing header using the timing schema. Forwarding metadata such as parent frontline and hop counts are only attached to downstream requests.

### Loop Prevention

The service tracks hop count via X-Unkey-Frontline-Hops and enforces a configurable maximum (default: 3). When a request exceeds MaxHops, it is rejected to prevent infinite routing loops.

### Connection Pooling

The service uses a shared http.Transport with conservative pooling and timeout settings to reduce latency by reusing connections safely.

### Error Handling

When proxying fails, the service writes clean JSON or HTML errors based on the client's Accept header and includes frontline identifiers for debugging.

## Constants

Header constants for frontline debugging and tracing
```go
const (
	// Headers set on BOTH response (to client) AND request (to downstream service)
	// These identify which frontline processed the request
	HeaderFrontlineID = "X-Unkey-Frontline-Id" // ID of the frontline instance
	HeaderRegion      = "X-Unkey-Region"       // Region of the frontline instance
	HeaderRequestID   = "X-Unkey-Request-Id"   // Request ID for tracing

	// Headers set ONLY on requests to downstream services (sentinel/remote frontline)
	// These provide additional context about the forwarding chain
	HeaderParentFrontlineID = "X-Unkey-Parent-Frontline-Id" // Frontline that forwarded this request
	HeaderParentRequestID   = "X-Unkey-Parent-Request-Id"   // Original request ID from parent frontline
	HeaderFrontlineHops     = "X-Unkey-Frontline-Hops"      // Number of frontline hops (loop prevention)
	HeaderDeploymentID      = "X-Deployment-Id"             // Deployment ID for local sentinel
	HeaderForwardedProto    = "X-Forwarded-Proto"           // Original protocol (https)
)
```


## Variables

requestStartTimeKey is the context key for storing the request start time. This is used to track timing across the request lifecycle without passing startTime as a parameter through multiple function calls.
```go
var requestStartTimeKey = zen.NewContextKey[time.Time]("request_start_time")
```


## Functions

### func ExtractHostname

```go
func ExtractHostname(host string) string
```

ExtractHostname extracts the hostname from a host string, stripping any port number. Examples:

  - "example.com:443" -> "example.com"
  - "example.com" -> "example.com"
  - "\[::1]:8080" -> "::1"
  - "192.168.1.1:80" -> "192.168.1.1"

### func RequestStartTimeFromContext

```go
func RequestStartTimeFromContext(ctx context.Context) (time.Time, bool)
```

RequestStartTimeFromContext retrieves the request start time from the context. Returns the start time and true if found, or zero time and false if not found.

### func WithRequestStartTime

```go
func WithRequestStartTime(ctx context.Context, startTime time.Time) context.Context
```

WithRequestStartTime stores the request start time in the context.


## Types

### type Config

```go
type Config struct {
	// FrontlineID is the current frontline instance ID
	FrontlineID string

	// Region is the current frontline region
	Region string

	// ApexDomain is the apex domain for remote NLB routing (e.g., "unkey.cloud")
	// Routes to frontline.{region}.{ApexDomain} (e.g., frontline.us-east-1.aws.unkey.cloud)
	ApexDomain string

	// Clock for time tracking
	Clock clock.Clock

	// MaxHops is the maximum number of frontline hops allowed before rejecting the request.
	// If 0, defaults to 3.
	MaxHops int

	// MaxIdleConns is the maximum number of idle connections to keep open.
	MaxIdleConns int

	// IdleConnTimeout is the maximum amount of time an idle connection will remain open.
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout is the maximum amount of time a TLS handshake will take.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout is the maximum amount of time to wait for response headers.
	ResponseHeaderTimeout time.Duration

	// Transport allows passing a shared HTTP transport for connection pooling
	// If nil, a new transport will be created with the other config values
	Transport *http.Transport
}
```

Config holds configuration for the proxy service.

### type Service

```go
type Service interface {
	// ForwardToSentinel forwards a request to a local sentinel service (HTTP)
	// Adds X-Unkey-Deployment-Id header for the sentinel to route to the correct deployment
	// Request start time is retrieved from context
	ForwardToSentinel(ctx context.Context, sess *zen.Session, sentinel *db.Sentinel, deploymentID string) error

	// ForwardToRegion forwards a request to a remote region (HTTPS)
	// Keeps the original hostname so the remote frontline can do TLS termination and routing
	// Request start time is retrieved from context
	ForwardToRegion(ctx context.Context, sess *zen.Session, targetRegion string) error
}
```

Service defines the interface for proxying requests to sentinels or remote NLBs.

