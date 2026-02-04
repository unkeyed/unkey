package wide

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestAlwaysSampler(t *testing.T) {
	sampler := AlwaysSampler{}
	ev := NewEventContext(EventConfig{})

	decision := sampler.ShouldSample(ev)
	assert.True(t, decision.Sample)
	assert.Equal(t, "always", decision.Reason)
}

func TestNeverSampler(t *testing.T) {
	sampler := NeverSampler{}
	ev := NewEventContext(EventConfig{})

	decision := sampler.ShouldSample(ev)
	assert.False(t, decision.Sample)
	assert.Equal(t, "never", decision.Reason)
}

func TestErrorSampler(t *testing.T) {
	sampler := ErrorSampler{}

	t.Run("samples errors", func(t *testing.T) {
		ev := NewEventContext(EventConfig{})
		ev.MarkError()

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
		assert.Equal(t, "error", decision.Reason)
	})

	t.Run("samples 5xx", func(t *testing.T) {
		ev := NewEventContext(EventConfig{})
		ev.Set(FieldStatusCode, 500)

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
		assert.Equal(t, "5xx_status", decision.Reason)

		ev2 := NewEventContext(EventConfig{})
		ev2.Set(FieldStatusCode, 503)
		decision = sampler.ShouldSample(ev2)
		assert.True(t, decision.Sample)
	})

	t.Run("does not sample success", func(t *testing.T) {
		ev := NewEventContext(EventConfig{})
		ev.Set(FieldStatusCode, 200)

		decision := sampler.ShouldSample(ev)
		assert.False(t, decision.Sample)
		assert.Equal(t, "no_error", decision.Reason)
	})
}

func TestSlowRequestSampler(t *testing.T) {
	sampler := SlowRequestSampler{Threshold: 500 * time.Millisecond}

	t.Run("samples slow requests", func(t *testing.T) {
		ev := NewEventContext(EventConfig{})
		// Simulate a slow request by creating an old context
		ev.start = time.Now().Add(-600 * time.Millisecond)

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
		assert.Equal(t, "slow", decision.Reason)
	})

	t.Run("does not sample fast requests", func(t *testing.T) {
		ev := NewEventContext(EventConfig{})

		decision := sampler.ShouldSample(ev)
		assert.False(t, decision.Sample)
		assert.Equal(t, "fast", decision.Reason)
	})
}

func TestRateSampler(t *testing.T) {
	t.Run("rate 1.0 always samples", func(t *testing.T) {
		sampler := RateSampler{Rate: 1.0}

		for i := 0; i < 100; i++ {
			ev := NewEventContext(EventConfig{})
			ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))

			decision := sampler.ShouldSample(ev)
			assert.True(t, decision.Sample)
		}
	})

	t.Run("rate 0 never samples", func(t *testing.T) {
		sampler := RateSampler{Rate: 0}

		for i := 0; i < 100; i++ {
			ev := NewEventContext(EventConfig{})
			ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))

			decision := sampler.ShouldSample(ev)
			assert.False(t, decision.Sample)
		}
	})

	t.Run("deterministic sampling", func(t *testing.T) {
		sampler := RateSampler{Rate: 0.5}
		requestID := uid.New(uid.RequestPrefix)

		ev := NewEventContext(EventConfig{})
		ev.Set(FieldRequestID, requestID)
		firstDecision := sampler.ShouldSample(ev)

		// Same request ID should produce same result
		for i := 0; i < 100; i++ {
			ev := NewEventContext(EventConfig{})
			ev.Set(FieldRequestID, requestID)

			decision := sampler.ShouldSample(ev)
			assert.Equal(t, firstDecision.Sample, decision.Sample)
		}
	})

	t.Run("sampling distribution", func(t *testing.T) {
		sampler := RateSampler{Rate: 0.5}
		sampled := 0
		total := 10000

		for i := 0; i < total; i++ {
			ev := NewEventContext(EventConfig{})
			ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))

			if sampler.ShouldSample(ev).Sample {
				sampled++
			}
		}

		rate := float64(sampled) / float64(total)
		assert.Greater(t, rate, 0.4, "sampling rate should be roughly 50%")
		assert.Less(t, rate, 0.6, "sampling rate should be roughly 50%")
	})
}

func TestCompositeSampler(t *testing.T) {
	t.Run("OR logic - samples if any sampler says yes", func(t *testing.T) {
		sampler := NewCompositeSampler(
			NeverSampler{},
			ErrorSampler{},
		)

		ev := NewEventContext(EventConfig{})
		ev.MarkError()

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
		assert.Equal(t, "error", decision.Reason)
	})

	t.Run("returns not_sampled if all say no", func(t *testing.T) {
		sampler := NewCompositeSampler(
			NeverSampler{},
			ErrorSampler{},
		)

		ev := NewEventContext(EventConfig{})
		ev.Set(FieldStatusCode, 200)

		decision := sampler.ShouldSample(ev)
		assert.False(t, decision.Sample)
		assert.Equal(t, "not_sampled", decision.Reason)
	})
}

func TestNewTailSampler(t *testing.T) {
	t.Run("samples errors", func(t *testing.T) {
		sampler := NewTailSampler(TailSamplerConfig{
			SuccessSampleRate: 0,
			SlowThreshold:     500 * time.Millisecond,
		})

		ev := NewEventContext(EventConfig{})
		ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))
		ev.MarkError()

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
	})

	t.Run("samples 5xx", func(t *testing.T) {
		sampler := NewTailSampler(TailSamplerConfig{
			SuccessSampleRate: 0,
			SlowThreshold:     500 * time.Millisecond,
		})

		ev := NewEventContext(EventConfig{})
		ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))
		ev.Set(FieldStatusCode, 500)

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
	})

	t.Run("samples slow requests", func(t *testing.T) {
		sampler := NewTailSampler(TailSamplerConfig{
			SuccessSampleRate: 0,
			SlowThreshold:     500 * time.Millisecond,
		})

		ev := NewEventContext(EventConfig{})
		ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))
		ev.start = time.Now().Add(-600 * time.Millisecond)

		decision := sampler.ShouldSample(ev)
		assert.True(t, decision.Sample)
	})

	t.Run("uses default config", func(t *testing.T) {
		config := DefaultTailSamplerConfig()
		assert.Equal(t, 0.01, config.SuccessSampleRate)
		assert.Equal(t, 500*time.Millisecond, config.SlowThreshold)
	})

	t.Run("normalizes invalid config", func(t *testing.T) {
		// Negative rate should become 0
		sampler := NewTailSampler(TailSamplerConfig{
			SuccessSampleRate: -0.5,
			SlowThreshold:     -1 * time.Second,
		})

		ev := NewEventContext(EventConfig{})
		ev.Set(FieldRequestID, uid.New(uid.RequestPrefix))
		ev.Set(FieldStatusCode, 200)

		// With rate 0 and threshold defaulted to 500ms, fast successful requests shouldn't be sampled
		decision := sampler.ShouldSample(ev)
		assert.False(t, decision.Sample)
	})
}
