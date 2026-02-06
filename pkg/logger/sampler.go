package logger

import (
	"math/rand/v2"
	"time"
)

// Sampler decides whether an event should be emitted when [End] is called.
// Implementations can inspect the completed event to make sampling decisions
// based on outcome, enabling tail sampling strategies.
type Sampler interface {
	Sample(event *Event) bool
}

// AlwaysSample emits every event unconditionally. Use during development
// or debugging when you need to see all log output. Not recommended for
// production due to log volume.
type AlwaysSample struct{}

// Sample always returns true, emitting every event.
func (AlwaysSample) Sample(*Event) bool {
	return true
}

// TailSampler provides probabilistic sampling with bias toward errors and
// slow requests. This is the recommended sampler for production: it reduces
// log volume for routine successes while ensuring errors and performance
// issues are always captured.
//
// Sampling rates are probabilities between 0.0 (never) and 1.0 (always).
// Events are evaluated in priority order: errors first, then slow requests,
// then baseline rate. An event matching multiple criteria (e.g., slow and
// has errors) is evaluated against the first matching rate.
type TailSampler struct {

	// SlowThreshold defines what duration qualifies as "slow". Events
	// exceeding this duration are always sampled.
	SlowThreshold time.Duration

	// SampleRate is the baseline probability for events that aren't errors
	// and aren't slow. Set to 0.1 to sample 10% of normal traffic.
	SampleRate float64
}

// Sample returns true if the event should be emitted based on configured rates.
// A single random value is generated and compared against each rate in order,
// so an event that matches multiple criteria still only has one chance to be
// sampled.
func (s TailSampler) Sample(event *Event) bool {
	rate := rand.Float64()

	if len(event.errors) > 0 {
		return true
	}

	if event.duration > s.SlowThreshold {
		return true
	}
	if rate < s.SampleRate {
		return true
	}
	return false
}
