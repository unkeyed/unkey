package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type RegionKey struct {
	Region string
}

type Outcome struct {
	NetErr           error
	StatusCode       int
	IsInfrastructure bool
}

func (o Outcome) IsFailure() bool {
	if o.NetErr != nil {
		return true
	}
	if !o.IsInfrastructure {
		return false
	}
	return o.StatusCode == 502 || o.StatusCode == 503 || o.StatusCode == 504
}

type Tracker interface {
	Allow(region string, now time.Time) bool
	Observe(region string, now time.Time, outcome Outcome)
}

type Config struct {
	Logger        logging.Logger
	Window        time.Duration
	MinRequests   int
	MaxErrorRate  float64
	MinErrorCount int
	Cooldown      time.Duration
	MaxEntries    int
	EntryTTL      time.Duration
}

func DefaultConfig(logger logging.Logger) Config {
	return Config{
		Logger:        logger,
		Window:        30 * time.Second,
		MinRequests:   20,
		MaxErrorRate:  0.5,
		MinErrorCount: 5,
		Cooldown:      20 * time.Second,
		MaxEntries:    512,
		EntryTTL:      30 * time.Minute,
	}
}

type entry struct {
	mu sync.Mutex

	windowStart   time.Time
	reqs          int
	errs          int
	openUntil     time.Time
	halfOpen      bool
	probeInFlight bool
	lastSeen      time.Time
}

type tracker struct {
	cfg     Config
	logger  logging.Logger
	mu      sync.RWMutex
	entries map[string]*entry
}

func NewTracker(cfg Config) Tracker {
	return &tracker{
		cfg:     cfg,
		logger:  cfg.Logger,
		mu:      sync.RWMutex{},
		entries: make(map[string]*entry),
	}
}

func (t *tracker) getOrCreate(region string, now time.Time) *entry {
	t.mu.RLock()
	e, exists := t.entries[region]
	t.mu.RUnlock()

	if exists {
		return e
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if e, exists = t.entries[region]; exists {
		return e
	}

	if len(t.entries) >= t.cfg.MaxEntries {
		t.evictOldest()
	}

	e = &entry{
		mu:            sync.Mutex{},
		windowStart:   now,
		reqs:          0,
		errs:          0,
		openUntil:     time.Time{},
		halfOpen:      false,
		probeInFlight: false,
		lastSeen:      now,
	}
	t.entries[region] = e
	return e
}

func (t *tracker) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, e := range t.entries {
		e.mu.Lock()
		if oldestKey == "" || e.lastSeen.Before(oldestTime) {
			oldestKey = k
			oldestTime = e.lastSeen
		}
		e.mu.Unlock()
	}

	if oldestKey != "" {
		delete(t.entries, oldestKey)
	}
}

func (t *tracker) Allow(region string, now time.Time) bool {
	e := t.getOrCreate(region, now)
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastSeen = now

	if now.Before(e.openUntil) {
		return false
	}

	if !e.openUntil.IsZero() && now.After(e.openUntil) && !e.halfOpen {
		e.halfOpen = true
		e.probeInFlight = false
	}

	if e.halfOpen {
		if e.probeInFlight {
			return false
		}
		e.probeInFlight = true
		t.logger.Info("circuit breaker half-open, allowing probe request",
			"region", region,
		)
	}

	return true
}

func (t *tracker) Observe(region string, now time.Time, outcome Outcome) {
	e := t.getOrCreate(region, now)
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastSeen = now
	isFailure := outcome.IsFailure()

	if e.halfOpen {
		if isFailure {
			e.openUntil = now.Add(t.cfg.Cooldown)
			e.halfOpen = false
			e.probeInFlight = false
			t.logger.Warn("circuit breaker probe failed, reopening",
				"region", region,
				"cooldown", t.cfg.Cooldown,
			)
		} else {
			e.halfOpen = false
			e.probeInFlight = false
			e.windowStart = now
			e.reqs = 0
			e.errs = 0
			e.openUntil = time.Time{}
			t.logger.Info("circuit breaker probe succeeded, closing",
				"region", region,
			)
		}
		return
	}

	if now.Sub(e.windowStart) > t.cfg.Window {
		e.windowStart = now
		e.reqs = 0
		e.errs = 0
	}

	e.reqs++
	if isFailure {
		e.errs++
	}

	errorRate := float64(e.errs) / float64(e.reqs)
	shouldTrip := (e.reqs >= t.cfg.MinRequests && errorRate >= t.cfg.MaxErrorRate) ||
		(e.errs >= t.cfg.MinErrorCount)

	if shouldTrip && e.openUntil.IsZero() {
		e.openUntil = now.Add(t.cfg.Cooldown)
		t.logger.Warn("circuit breaker opened",
			"region", region,
			"reqs", e.reqs,
			"errs", e.errs,
			"errorRate", errorRate,
			"cooldown", t.cfg.Cooldown,
		)
	}
}

type ctxKey struct{}

func WithKey(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, ctxKey{}, region)
}

func KeyFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKey{}).(string)
	return v, ok
}
