package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/circuitbreaker/metrics"
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

	// failureRatio, when > 0, switches tripping from an absolute count to a
	// rate: the circuit opens once failures/requests within the cyclic period
	// reaches this fraction, but only after at least minRequests requests so a
	// tiny sample can't trip it. This is robust to traffic volume — a brief
	// sub-1% blip never trips, while a real outage (most requests failing)
	// trips fast regardless of throughput. When 0, tripThreshold is used.
	failureRatio float64

	// minRequests is the minimum number of requests within the cyclic period
	// before failureRatio is evaluated. Ignored when failureRatio is 0.
	minRequests int

	// Clock to use for timing, defaults to the system clock but can be overridden for testing
	clock clock.Clock
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
// where this function returns true count toward the trip threshold. Ignored
// errors are neutral: they do not count as failures or successes. By default,
// all non-nil errors except context.Canceled are considered downstream errors.
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

// WithFailureRatio switches tripping from an absolute failure count to a
// failure rate. The circuit opens once failures/requests within the cyclic
// period reaches ratio (0..1], but only after at least minRequests requests so
// a small sample can't trip it. Prefer this for high-throughput call sites
// where a fixed count is either trivially exceeded by harmless blips or never
// reached during a low-traffic outage. ratio <= 0 disables it (falls back to
// WithTripThreshold). A ratio above 1 is clamped to 1 (requires 100% failures)
// so a typo can't silently produce a breaker that never trips on rate.
func WithFailureRatio(ratio float64, minRequests int) applyConfig {
	return func(c *config) {
		if ratio > 1 {
			ratio = 1
		}
		c.failureRatio = ratio
		c.minRequests = minRequests
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
			return err != nil && !errors.Is(err, context.Canceled)
		},
		tripThreshold: 5,
		failureRatio:  0,
		minRequests:   0,
		clock:         clock.New(),
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

	metrics.CircuitBreakerRequests.WithLabelValues(cb.config.name, string(cb.state)).Inc()

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

	isDownstreamError := err != nil && cb.config.isDownstreamError(err)
	if err != nil && !isDownstreamError {
		return
	}

	cb.requests++
	switch {
	case isDownstreamError:
		cb.failures++
		cb.consecutiveFailures++
		cb.consecutiveSuccesses = 0
	case err == nil:
		cb.successes++
		cb.consecutiveSuccesses++
		cb.consecutiveFailures = 0
	}

	switch cb.state {

	case Closed:
		if cb.shouldTrip() {
			cb.state = Open
		}

	case HalfOpen:
		if cb.consecutiveSuccesses >= cb.config.maxRequests {
			cb.state = Closed
		}
	}

}

// shouldTrip reports whether the accumulated counts in the current cyclic
// period warrant opening the circuit. Rate-based when failureRatio is set,
// otherwise the legacy absolute count. Caller must hold the lock.
func (cb *CB[Res]) shouldTrip() bool {
	if cb.config.failureRatio > 0 {
		if cb.requests < cb.config.minRequests {
			return false
		}
		return float64(cb.failures)/float64(cb.requests) >= cb.config.failureRatio
	}
	return cb.failures >= cb.config.tripThreshold
}
