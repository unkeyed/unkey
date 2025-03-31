// Package circuitbreaker implements the Circuit Breaker pattern to prevent
// cascading failures when dependent services are unavailable or experiencing
// high latency.
//
// The circuit breaker monitors for failures and, once a threshold is reached,
// "trips" into an open state to fail fast and reduce load on the struggling
// service. After a timeout, the circuit transitions to a half-open state to
// test if the service has recovered before fully closing the circuit again.
//
// The implementation supports:
// - Configurable failure thresholds
// - Custom backoff strategies
// - Automatic transition between Open, Closed, and HalfOpen states
// - Generic response types for type-safe usage
//
// Example usage:
//
//	// Create a circuit breaker for HTTP requests
//	cb := circuitbreaker.New[*http.Response]("api_service",
//	    circuitbreaker.WithTripThreshold(5),
//	    circuitbreaker.WithTimeout(10 * time.Second),
//	)
//
//	// Use the circuit breaker
//	resp, err := cb.Do(ctx, func(ctx context.Context) (*http.Response, error) {
//	    return http.Get("https://api.example.com/data")
//	})
//
//	if err != nil {
//	    if errors.Is(err, circuitbreaker.ErrTripped) {
//	        // Circuit is open, fail fast without hitting the service
//	        return fallbackResponse()
//	    }
//	    // Other error occurred during the request
//	    return handleError(err)
//	}
//
//	// Process the successful response
//	return processResponse(resp)
package circuitbreaker
