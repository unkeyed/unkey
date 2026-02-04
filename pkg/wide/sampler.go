package wide

import (
	"hash/fnv"
	"time"
)

// Sampler determines whether an event should be logged.
// Implement this interface to create custom sampling strategies.
//
// The sampler receives the full EventContext and can inspect any fields
// (e.g., status_code, error, duration) to make its decision.
type Sampler interface {
	// ShouldSample returns a SampleDecision indicating whether to log and why.
	ShouldSample(ev *EventContext) SampleDecision
}

// SampleDecision contains the result of a sampling decision along with the reason.
type SampleDecision struct {
	// Sample is true if the event should be logged.
	Sample bool

	// Reason explains why the sampling decision was made.
	Reason string
}

// --- Always Sampler ---

// AlwaysSampler always samples every event. Useful for debugging or low-traffic services.
type AlwaysSampler struct{}

// ShouldSample always returns true.
func (s AlwaysSampler) ShouldSample(_ *EventContext) SampleDecision {
	return SampleDecision{Sample: true, Reason: "always"}
}

// --- Never Sampler ---

// NeverSampler never samples any event. Useful for disabling logging entirely.
type NeverSampler struct{}

// ShouldSample always returns false.
func (s NeverSampler) ShouldSample(_ *EventContext) SampleDecision {
	return SampleDecision{Sample: false, Reason: "never"}
}

// --- Error Sampler ---

// ErrorSampler samples events that have errors or 5xx status codes.
// For HTTP services, set FieldStatusCode before calling Emit().
type ErrorSampler struct{}

// ShouldSample returns true if the event has an error or 5xx status.
func (s ErrorSampler) ShouldSample(ev *EventContext) SampleDecision {
	if ev.HasError() {
		return SampleDecision{Sample: true, Reason: "error"}
	}

	// Check status code if present (for HTTP services)
	if statusCode, ok := ev.Get(FieldStatusCode); ok {
		if code, ok := statusCode.(int); ok && code >= 500 {
			return SampleDecision{Sample: true, Reason: "5xx_status"}
		}
	}

	return SampleDecision{Sample: false, Reason: "no_error"}
}

// --- Slow Request Sampler ---

// SlowRequestSampler samples events that exceed a duration threshold.
type SlowRequestSampler struct {
	Threshold time.Duration
}

// ShouldSample returns true if the request duration exceeds the threshold.
func (s SlowRequestSampler) ShouldSample(ev *EventContext) SampleDecision {
	if ev.Duration() >= s.Threshold {
		return SampleDecision{Sample: true, Reason: "slow"}
	}
	return SampleDecision{Sample: false, Reason: "fast"}
}

// --- Rate Sampler ---

// RateSampler samples a percentage of events using deterministic hashing.
type RateSampler struct {
	Rate float64 // 0.0 to 1.0
}

// ShouldSample returns true based on deterministic hash of request ID.
func (s RateSampler) ShouldSample(ev *EventContext) SampleDecision {
	if s.Rate >= 1.0 {
		return SampleDecision{Sample: true, Reason: "sampled"}
	}
	if s.Rate <= 0 {
		return SampleDecision{Sample: false, Reason: "not_sampled"}
	}

	requestID, _ := ev.Get(FieldRequestID)
	reqIDStr, _ := requestID.(string)

	// Use FNV-1a hash for fast, good distribution
	h := fnv.New64a()
	h.Write([]byte(reqIDStr))
	hash := h.Sum64()

	// Convert hash to a value between 0 and 1
	hashFloat := float64(hash) / float64(^uint64(0))

	if hashFloat < s.Rate {
		return SampleDecision{Sample: true, Reason: "sampled"}
	}
	return SampleDecision{Sample: false, Reason: "not_sampled"}
}

// --- Composite Sampler ---

// CompositeSampler combines multiple samplers with OR logic.
// If any sampler says to sample, the event is sampled.
type CompositeSampler struct {
	samplers []Sampler
}

// NewCompositeSampler creates a sampler that combines multiple samplers.
// The event is sampled if ANY sampler returns true (OR logic).
func NewCompositeSampler(samplers ...Sampler) *CompositeSampler {
	return &CompositeSampler{samplers: samplers}
}

// ShouldSample returns true if any of the underlying samplers return true.
func (s *CompositeSampler) ShouldSample(ev *EventContext) SampleDecision {
	for _, sampler := range s.samplers {
		decision := sampler.ShouldSample(ev)
		if decision.Sample {
			return decision
		}
	}
	return SampleDecision{Sample: false, Reason: "not_sampled"}
}

// --- Tail Sampler (convenience constructor) ---

// TailSamplerConfig holds configuration for creating a standard tail sampler.
type TailSamplerConfig struct {
	// SuccessSampleRate is the sampling rate for successful requests (0.0 - 1.0).
	// Default is 0.01 (1%).
	SuccessSampleRate float64

	// SlowThreshold is the duration above which a request is considered slow.
	// Slow requests are always sampled. Default is 500ms.
	SlowThreshold time.Duration
}

// DefaultTailSamplerConfig returns a TailSamplerConfig with sensible defaults.
func DefaultTailSamplerConfig() TailSamplerConfig {
	return TailSamplerConfig{
		SuccessSampleRate: 0.01, // 1%
		SlowThreshold:     500 * time.Millisecond,
	}
}

// NewTailSampler creates a composite sampler that implements tail-based sampling:
//   - Always sample: errors, 5xx status codes, slow requests
//   - Sample at configured rate: everything else
func NewTailSampler(config TailSamplerConfig) Sampler {
	rate := config.SuccessSampleRate
	if rate < 0 {
		rate = 0
	} else if rate > 1 {
		rate = 1
	}

	threshold := config.SlowThreshold
	if threshold <= 0 {
		threshold = 500 * time.Millisecond
	}

	return NewCompositeSampler(
		ErrorSampler{},
		SlowRequestSampler{Threshold: threshold},
		RateSampler{Rate: rate},
	)
}
