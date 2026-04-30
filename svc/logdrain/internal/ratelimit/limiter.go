// Package ratelimit provides rate limiting and back-pressure mechanisms for
// logdrain sinks to prevent catch-up storms and provider overload during
// recovery scenarios.
package ratelimit

import (
	"context"
	"math"
	"math/rand"
	"time"

	"golang.org/x/time/rate"
)

// DrainLimiter provides per-drain rate limiting with exponential backoff
// and jitter to prevent thundering herds during provider recovery.
type DrainLimiter struct {
	limiter          *rate.Limiter
	consecutiveErrs  int
	lastSuccessTime  time.Time
	baseInterval     time.Duration
	maxInterval      time.Duration
	backoffMultiple  float64
}

// NewDrainLimiter creates a rate limiter for a single drain
func NewDrainLimiter(rps int) *DrainLimiter {
	return &DrainLimiter{
		limiter:         rate.NewLimiter(rate.Limit(rps), rps*2), // 2x burst
		consecutiveErrs: 0,
		lastSuccessTime: time.Now(),
		baseInterval:    time.Second,
		maxInterval:     5 * time.Minute,
		backoffMultiple: 1.5,
	}
}

// Wait blocks until the rate limit allows the request, implementing
// exponential backoff with jitter based on recent error history.
func (d *DrainLimiter) Wait(ctx context.Context) error {
	// First check rate limit
	if err := d.limiter.Wait(ctx); err != nil {
		return err
	}

	// Apply exponential backoff if there have been recent errors
	if d.consecutiveErrs > 0 {
		backoffDuration := d.calculateBackoff()
		
		select {
		case <-time.After(backoffDuration):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// RecordSuccess resets the error count and backoff
func (d *DrainLimiter) RecordSuccess() {
	d.consecutiveErrs = 0
	d.lastSuccessTime = time.Now()
}

// RecordError increments the error count for backoff calculation
func (d *DrainLimiter) RecordError() {
	d.consecutiveErrs++
}

// calculateBackoff computes exponential backoff duration with jitter
func (d *DrainLimiter) calculateBackoff() time.Duration {
	if d.consecutiveErrs == 0 {
		return 0
	}

	// Exponential backoff: base * multiplier^errors
	backoff := float64(d.baseInterval) * math.Pow(d.backoffMultiple, float64(d.consecutiveErrs-1))
	
	// Cap at max interval
	if backoff > float64(d.maxInterval) {
		backoff = float64(d.maxInterval)
	}

	// Add jitter (±25%)
	jitterRange := backoff * 0.25
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange
	
	duration := time.Duration(backoff + jitter)
	if duration < 0 {
		duration = d.baseInterval
	}

	return duration
}

// GetConsecutiveErrors returns the current consecutive error count
func (d *DrainLimiter) GetConsecutiveErrors() int {
	return d.consecutiveErrs
}

// TimeSinceLastSuccess returns how long since the last successful operation
func (d *DrainLimiter) TimeSinceLastSuccess() time.Duration {
	return time.Since(d.lastSuccessTime)
}

// SlowStartLimiter implements catch-up storm prevention by gradually
// increasing batch sizes after high lag periods.
type SlowStartLimiter struct {
	normalBatchSize int
	currentSize     int
	minSize         int
	lagThreshold    time.Duration
}

// NewSlowStartLimiter creates a limiter for batch size management
func NewSlowStartLimiter(normalSize int, lagThreshold time.Duration) *SlowStartLimiter {
	return &SlowStartLimiter{
		normalBatchSize: normalSize,
		currentSize:     normalSize,
		minSize:         max(10, normalSize/10),
		lagThreshold:    lagThreshold,
	}
}

// GetBatchSize returns the current allowed batch size based on lag
func (s *SlowStartLimiter) GetBatchSize(lag time.Duration) int {
	if lag > s.lagThreshold {
		// High lag - use minimum batch size
		s.currentSize = s.minSize
	} else {
		// Gradually increase back to normal
		s.currentSize = min(s.normalBatchSize, s.currentSize*2)
	}
	
	return s.currentSize
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
