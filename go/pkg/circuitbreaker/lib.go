package circuitbreaker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

// CB implements the CircuitBreaker interface with configurable failure detection
// and recovery behavior.
type CB[Res any] struct {
	sync.Mutex
	// This is a pointer to the configuration of the circuit breaker because we
	// need to modify the clock for testing
	config *config

	logger logging.Logger

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

// config holds the configuration parameters for a circuit breaker instance.
type config struct {
	name string
	// Max requests that may pass through the circuit breaker in its half-open state
	// If all requests are successful, the circuit will close
	// If any request fails, the circuit will remaing half open until the next cycle
	maxRequests int

	// Interval to clear counts while the circuit is closed
	cyclicPeriod time.Duration

	// How long the circuit will stay open before transitioning to half-open
	timeout time.Duration

	// Determine whether the error is a downstream error or not
	// If the error is a downstream error, the circuit will count it
	// If the error is not a downstream error, the circuit will not count it
	isDownstreamError func(error) bool

	// How many downstream errors within a cyclic period are allowed before the
	// circuit trips and opens
	tripThreshold int

	// Clock to use for timing, defaults to the system clock but can be overridden for testing
	clock clock.Clock

	logger logging.Logger
}

// WithMaxRequests configures the maximum number of requests allowed during
// the half-open state. Once this threshold is reached, the circuit will either
// close (if all requests succeeded) or remain open (if any request failed).
//
// Default: 10
func WithMaxRequests(maxRequests int) applyConfig {
	return func(c *config) {
		c.maxRequests = maxRequests
	}
}

// WithCyclicPeriod sets the interval at which request counters are reset
// while the circuit is closed. This determines how frequently the circuit
// "forgets" about past failures.
//
// Default: 5 seconds
func WithCyclicPeriod(cyclicPeriod time.Duration) applyConfig {
	return func(c *config) {
		c.cyclicPeriod = cyclicPeriod
	}
}

// WithIsDownstreamError provides a function to determine if an error should
// be counted towards the failure threshold. This allows the circuit breaker
// to ignore certain types of errors (e.g., validation errors) that don't
// indicate a problem with the downstream service.
//
// Default: func(err error) bool { return err != nil }
func WithIsDownstreamError(isDownstreamError func(error) bool) applyConfig {
	return func(c *config) {
		c.isDownstreamError = isDownstreamError
	}
}

// WithTripThreshold sets how many failures must occur within a cyclic period
// before the circuit breaker trips and enters the open state.
//
// Default: 5
func WithTripThreshold(tripThreshold int) applyConfig {
	return func(c *config) {
		c.tripThreshold = tripThreshold
	}
}

// WithTimeout sets how long the circuit breaker stays in the open state
// before transitioning to half-open to test if the service has recovered.
//
// Default: 1 minute
func WithTimeout(timeout time.Duration) applyConfig {
	return func(c *config) {
		c.timeout = timeout
	}
}

// WithClock provides a custom clock implementation for testing time-based behavior.
// This should only be used in test code.
func WithClock(clock clock.Clock) applyConfig {
	return func(c *config) {
		c.clock = clock
	}
}

// WithLogger provides a custom logger for the circuit breaker to use.
// If not provided, a no-op logger will be used.
func WithLogger(logger logging.Logger) applyConfig {
	return func(c *config) {
		c.logger = logger
	}
}

// applyConfig applies a config setting to the circuit breaker
type applyConfig func(*config)

// New creates a new circuit breaker with configurable behavior.
// The name parameter identifies this circuit breaker for logging and metrics.
// The generic parameter Res specifies the response type from the protected operation.
//
// Configuration is provided via functional options:
//
//	cb := New[*http.Response]("api_service",
//	    WithTripThreshold(10),
//	    WithTimeout(30 * time.Second),
//	    WithIsDownstreamError(func(err error) bool {
//	        // Only count 5xx errors as downstream failures
//	        var httpErr *HttpError
//	        if errors.As(err, &httpErr) {
//	            return httpErr.StatusCode >= 500
//	        }
//	        return err != nil
//	    }),
//	)
func New[Res any](name string, applyConfigs ...applyConfig) *CB[Res] {

	cfg := &config{
		name:         name,
		maxRequests:  10,
		cyclicPeriod: 5 * time.Second,
		timeout:      time.Minute,
		isDownstreamError: func(err error) bool {
			return err != nil
		},
		tripThreshold: 5,
		clock:         clock.New(),
		logger:        logging.NewNoop(),
	}

	for _, apply := range applyConfigs {
		apply(cfg)
	}

	cb := &CB[Res]{
		Mutex:                sync.Mutex{},
		config:               cfg,
		logger:               cfg.logger,
		state:                Closed,
		resetCountersAt:      cfg.clock.Now().Add(cfg.cyclicPeriod),
		resetStateAt:         cfg.clock.Now().Add(cfg.timeout),
		requests:             0,
		successes:            0,
		failures:             0,
		consecutiveSuccesses: 0,
		consecutiveFailures:  0,
	}

	return cb
}

var _ CircuitBreaker[any] = (*CB[any])(nil)

// Do executes a function with circuit breaker protection.
// If the circuit is open, it returns ErrTripped without executing the function.
// If the circuit is half-open and the maximum test requests are already in flight,
// it returns ErrTooManyRequests.
//
// The function is wrapped with appropriate tracing to track circuit breaker
// operations in observability systems.
func (cb *CB[Res]) Do(ctx context.Context, fn func(context.Context) (Res, error)) (res Res, err error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("circuitbreaker.%s.Do", cb.config.name))
	defer span.End()

	err = cb.preflight(ctx)
	if err != nil {
		return res, err
	}

	ctx, fnSpan := tracing.Start(ctx, fmt.Sprintf("circuitbreaker.%s.fn", cb.config.name))
	res, err = fn(ctx)
	fnSpan.End()

	cb.postflight(ctx, err)

	return res, err

}

// preflight checks if the circuit is ready to accept a request.
// It updates internal counters and state based on configured intervals.
func (cb *CB[Res]) preflight(ctx context.Context) error {
	_, span := tracing.Start(ctx, fmt.Sprintf("circuitbreaker.%s.preflight", cb.config.name))
	defer span.End()
	cb.Lock()
	defer cb.Unlock()

	now := cb.config.clock.Now()

	if now.After(cb.resetCountersAt) {
		cb.requests = 0
		cb.successes = 0
		cb.failures = 0
		cb.consecutiveSuccesses = 0
		cb.consecutiveFailures = 0
		cb.resetCountersAt = now.Add(cb.config.cyclicPeriod)
	}
	if cb.state == Open && now.After(cb.resetStateAt) {
		cb.state = HalfOpen
		cb.resetStateAt = now.Add(cb.config.timeout)
	}

	if cb.state == Open {
		return ErrTripped
	}

	cb.logger.Debug("circuit breaker state",
		slog.String("state", string(cb.state)),
		slog.Int("requests", cb.requests),
		slog.Int("maxRequests", cb.config.maxRequests),
	)
	if cb.state == HalfOpen && cb.requests >= cb.config.maxRequests {
		return ErrTooManyRequests
	}
	return nil
}

// postflight updates the circuit breaker state based on the result of the request.
// It tracks successes and failures to determine whether to open or close the circuit.
func (cb *CB[Res]) postflight(ctx context.Context, err error) {
	_, span := tracing.Start(ctx, fmt.Sprintf("circuitbreaker.%s.postflight", cb.config.name))
	defer span.End()
	cb.Lock()
	defer cb.Unlock()
	cb.requests++
	if cb.config.isDownstreamError(err) {
		cb.failures++
		cb.consecutiveFailures++
		cb.consecutiveSuccesses = 0
	} else {
		cb.successes++
		cb.consecutiveSuccesses++
		cb.consecutiveFailures = 0
	}

	switch cb.state {

	case Closed:
		if cb.failures >= cb.config.tripThreshold {
			cb.state = Open
		}

	case HalfOpen:
		if cb.consecutiveSuccesses >= cb.config.maxRequests {
			cb.state = Closed
		}
	}

}
