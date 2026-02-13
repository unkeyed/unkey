package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
)

// CB is the concrete implementation of [CircuitBreaker]. It tracks request
// success and failure counts to determine when to trip the circuit. CB is
// safe for concurrent use; all state mutations are protected by an embedded
// mutex. Use [New] to create a properly initialized instance.
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

	metrics Metrics
}

// WithMaxRequests sets the maximum number of requests allowed through during
// the [HalfOpen] state. If all probe requests succeed, the circuit closes.
// Defaults to 10.
func WithMaxRequests(maxRequests int) applyConfig {
	return func(c *config) {
		c.maxRequests = maxRequests
	}
}

// WithCyclicPeriod sets the interval at which failure counts are reset while
// the circuit is [Closed]. A shorter period makes the circuit less sensitive
// to sporadic failures. Defaults to 5 seconds.
func WithCyclicPeriod(cyclicPeriod time.Duration) applyConfig {
	return func(c *config) {
		c.cyclicPeriod = cyclicPeriod
	}
}

// WithIsDownstreamError provides a function to classify errors. Only errors
// where this function returns true count toward the trip threshold. By default,
// all non-nil errors are considered downstream errors.
func WithIsDownstreamError(isDownstreamError func(error) bool) applyConfig {
	return func(c *config) {
		c.isDownstreamError = isDownstreamError
	}
}

// WithTripThreshold sets the number of failures within a cyclic period that
// will cause the circuit to trip from [Closed] to [Open]. Defaults to 5.
func WithTripThreshold(tripThreshold int) applyConfig {
	return func(c *config) {
		c.tripThreshold = tripThreshold
	}
}

// WithTimeout sets how long the circuit remains [Open] before transitioning
// to [HalfOpen] to probe for recovery. Defaults to 1 minute.
func WithTimeout(timeout time.Duration) applyConfig {
	return func(c *config) {
		c.timeout = timeout
	}
}

// WithClock sets a custom clock for timing operations. This is primarily
// useful for testing to control time progression.
func WithClock(clock clock.Clock) applyConfig {
	return func(c *config) {
		c.clock = clock
	}
}

// WithMetrics sets the metrics instance used to record circuit breaker
// observations. When nil, no metrics are emitted.
func WithMetrics(m Metrics) applyConfig {
	return func(c *config) {
		c.metrics = m
	}
}

// applyConfig is a functional option for configuring a circuit breaker.
// Use the With* functions to create options.
type applyConfig func(*config)

// New creates a new circuit breaker with the given name and configuration
// options. The name is used for metrics and tracing identification. The
// circuit breaker starts in the [Closed] state, allowing all requests through.
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
		metrics:       nil,
	}

	for _, apply := range applyConfigs {
		apply(cfg)
	}

	cb := &CB[Res]{
		Mutex:                sync.Mutex{},
		config:               cfg,
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

var _ CircuitBreaker[any] = &CB[any]{
	Mutex:                sync.Mutex{},
	config:               nil,
	state:                Closed,
	resetCountersAt:      time.Time{},
	resetStateAt:         time.Time{},
	requests:             0,
	successes:            0,
	failures:             0,
	consecutiveSuccesses: 0,
	consecutiveFailures:  0,
}

// Do executes fn if the circuit allows it. Returns [ErrTripped] immediately
// if the circuit is [Open], or [ErrTooManyRequests] if in [HalfOpen] state
// and the probe limit is exceeded. On success or failure, the result is
// recorded to update the circuit state. The zero value of Res is returned
// when the circuit rejects the request.
func (cb *CB[Res]) Do(ctx context.Context, fn func(context.Context) (Res, error)) (res Res, err error) {
	err = cb.preflight(ctx)
	if err != nil {
		return res, err
	}

	res, err = fn(ctx)

	cb.postflight(ctx, err)

	return res, err
}

// preflight checks if the circuit is ready to accept a request
func (cb *CB[Res]) preflight(_ context.Context) error {
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

	if cb.config.metrics != nil {
		cb.config.metrics.RecordRequest(cb.config.name, string(cb.state))
	}

	if cb.state == Open {
		return ErrTripped
	}

	logger.Debug("circuit breaker state", "state", string(cb.state), "requests", cb.requests, "maxRequests", cb.config.maxRequests)
	if cb.state == HalfOpen && cb.requests >= cb.config.maxRequests {
		return ErrTooManyRequests
	}
	return nil
}

// postflight updates the circuit breaker state based on the result of the request
func (cb *CB[Res]) postflight(_ context.Context, err error) {
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
