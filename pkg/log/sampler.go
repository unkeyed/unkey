package log

import (
	"math/rand"
	"time"
)

// Sampler decides whether an event should be emitted after it completes.
type Sampler interface {
	Sample(event *Event) bool
}

// AlwaysSample emits every event. Use for development or debugging.
type AlwaysSample struct{}

func (AlwaysSample) Sample(*Event) bool {
	return true
}

// TailSampler provides probabilistic sampling with bias toward errors and slow requests.
// Sampling rates are values between 0 and 1 representing probability of emission.
type TailSampler struct {
	// ErrorSampleRate is the probability of emitting events with errors.
	ErrorSampleRate float64
	// SlowThreshold defines what duration qualifies as "slow".
	SlowThreshold time.Duration
	// SlowSampleRate is the probability of emitting slow events.
	SlowSampleRate float64
	// SampleRate is the baseline probability for all other events.
	SampleRate float64
	// Rand is the random source for sampling decisions.
	Rand *rand.Rand
}

// Sample returns true if the event should be emitted based on configured rates.
// Events are evaluated in order: errors, slow requests, then baseline rate.
func (s TailSampler) Sample(event *Event) bool {
	rate := s.Rand.Float64()

	if len(event.errors) > 0 && rate < s.ErrorSampleRate {
		return true
	}

	if event.duration > s.SlowThreshold && rate < s.SlowSampleRate {
		return true
	}
	if rate < s.SampleRate {
		return true
	}
	return false
}
