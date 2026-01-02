package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

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

func WithMaxRequests(maxRequests int) applyConfig {
	return func(c *config) {
		c.maxRequests = maxRequests
	}
}

func WithCyclicPeriod(cyclicPeriod time.Duration) applyConfig {
	return func(c *config) {
		c.cyclicPeriod = cyclicPeriod
	}
}
func WithIsDownstreamError(isDownstreamError func(error) bool) applyConfig {
	return func(c *config) {
		c.isDownstreamError = isDownstreamError
	}
}
func WithTripThreshold(tripThreshold int) applyConfig {
	return func(c *config) {
		c.tripThreshold = tripThreshold
	}
}

func WithTimeout(timeout time.Duration) applyConfig {
	return func(c *config) {
		c.timeout = timeout
	}
}

// for testing
func WithClock(clock clock.Clock) applyConfig {
	return func(c *config) {
		c.clock = clock
	}
}

func WithLogger(logger logging.Logger) applyConfig {
	return func(c *config) {
		c.logger = logger
	}
}

// applyConfig applies a config setting to the circuit breaker
type applyConfig func(*config)

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

var _ CircuitBreaker[any] = &CB[any]{
	Mutex:                sync.Mutex{},
	config:               nil,
	logger:               nil,
	state:                Closed,
	resetCountersAt:      time.Time{},
	resetStateAt:         time.Time{},
	requests:             0,
	successes:            0,
	failures:             0,
	consecutiveSuccesses: 0,
	consecutiveFailures:  0,
}

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

// preflight checks if the circuit is ready to accept a request
func (cb *CB[Res]) preflight(ctx context.Context) error {
	_, span := tracing.Start(ctx, fmt.Sprintf("circuitbreaker.%s.preflight", cb.config.name)) // nolint:ineffassign // Context is used by tracing
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

	metrics.CircuitBreakerRequests.WithLabelValues(cb.config.name, string(cb.state)).Inc()

	if cb.state == Open {
		return ErrTripped
	}

	cb.logger.Debug("circuit breaker state", "state", string(cb.state), "requests", cb.requests, "maxRequests", cb.config.maxRequests)
	if cb.state == HalfOpen && cb.requests >= cb.config.maxRequests {
		return ErrTooManyRequests
	}
	return nil
}

// postflight updates the circuit breaker state based on the result of the request
func (cb *CB[Res]) postflight(ctx context.Context, err error) {
	_, span := tracing.Start(ctx, fmt.Sprintf("circuitbreaker.%s.postflight", cb.config.name)) // nolint:ineffassign // Context is used by tracing
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
