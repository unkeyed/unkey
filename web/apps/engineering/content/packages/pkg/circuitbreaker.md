---
title: circuitbreaker
description: "implements the Circuit Breaker pattern to prevent"
---

Package circuitbreaker implements the Circuit Breaker pattern to prevent cascading failures when dependent services are unavailable or experiencing high latency.

The circuit breaker monitors for failures and, once a threshold is reached, "trips" into an open state to fail fast and reduce load on the struggling service. After a timeout, the circuit transitions to a half-open state to test if the service has recovered before fully closing the circuit again.

The implementation supports: - Configurable failure thresholds - Custom backoff strategies - Automatic transition between Open, Closed, and HalfOpen states - Generic response types for type-safe usage

Example usage:

	// Create a circuit breaker for HTTP requests
	cb := circuitbreaker.New[*http.Response]("api_service",
	    circuitbreaker.WithTripThreshold(5),
	    circuitbreaker.WithTimeout(10 * time.Second),
	)

	// Use the circuit breaker
	resp, err := cb.Do(ctx, func(ctx context.Context) (*http.Response, error) {
	    return http.Get("https://api.example.com/data")
	})

	if err != nil {
	    if errors.Is(err, circuitbreaker.ErrTripped) {
	        // Circuit is open, fail fast without hitting the service
	        return fallbackResponse()
	    }
	    // Other error occurred during the request
	    return handleError(err)
	}

	// Process the successful response
	return processResponse(resp)

## Variables

```go
var (
	// ErrTripped is returned when the circuit breaker is in the [Open] state and
	// is rejecting requests to protect the downstream service. Callers should
	// implement fallback logic or return an appropriate error to their clients.
	ErrTripped = errors.New("circuit breaker is open")

	// ErrTooManyRequests is returned when the circuit breaker is in the [HalfOpen]
	// state and has already allowed the maximum number of probe requests through.
	// The caller should wait for the current probe requests to complete before
	// retrying.
	ErrTooManyRequests = errors.New("too many requests during half open state")
)
```


## Functions

### func IsErrTooManyRequests

```go
func IsErrTooManyRequests(err error) bool
```

IsErrTooManyRequests reports whether err is or wraps \[ErrTooManyRequests].

### func IsErrTripped

```go
func IsErrTripped(err error) bool
```

IsErrTripped reports whether err is or wraps \[ErrTripped].


## Types

### type CB

```go
type CB[Res any] struct {
	sync.Mutex
	// This is a pointer to the configuration of the circuit breaker because we
	// need to modify the clock for testing
	config *config

	// State of the circuit
	state State

	// reset the counters every cyclic period
	resetCountersAt time.Time

	// reset the state every recoveryTimeout
	resetStateAt time.Time

	// counters are protected by the mutex and are reset every cyclic period
	requests             int
	successes            int
	failures             int
	consecutiveSuccesses int
	consecutiveFailures  int
}
```

CB is the concrete implementation of \[CircuitBreaker]. It tracks request success and failure counts to determine when to trip the circuit. CB is safe for concurrent use; all state mutations are protected by an embedded mutex. Use \[New] to create a properly initialized instance.

#### func New

```go
func New[Res any](name string, applyConfigs ...applyConfig) *CB[Res]
```

New creates a new circuit breaker with the given name and configuration options. The name is used for metrics and tracing identification. The circuit breaker starts in the \[Closed] state, allowing all requests through.

#### func (CB) Do

```go
func (cb *CB[Res]) Do(ctx context.Context, fn func(context.Context) (Res, error)) (res Res, err error)
```

Do executes fn if the circuit allows it. Returns \[ErrTripped] immediately if the circuit is \[Open], or \[ErrTooManyRequests] if in \[HalfOpen] state and the probe limit is exceeded. On success or failure, the result is recorded to update the circuit state. The zero value of Res is returned when the circuit rejects the request.

### type CircuitBreaker

```go
type CircuitBreaker[Res any] interface {
	// Do executes the provided function if the circuit is closed or allows a
	// probe request in half-open state. Returns [ErrTripped] if the circuit is
	// open, or [ErrTooManyRequests] if the half-open probe limit is exceeded.
	Do(ctx context.Context, fn func(context.Context) (Res, error)) (Res, error)
}
```

CircuitBreaker wraps operations that call downstream services and provides automatic failure detection and recovery. When failures exceed a threshold, the circuit "opens" to fail fast and reduce load on the struggling service.

### type State

```go
type State string
```

State represents the current state of a circuit breaker. The circuit breaker transitions between states based on the success or failure of requests: \[Closed] -> \[Open] -> \[HalfOpen] -> \[Closed] (or back to \[Open] on failure).

