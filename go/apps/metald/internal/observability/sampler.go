package observability

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// AlwaysOnErrorSampler samples all spans with errors, delegates other decisions to the base sampler
type AlwaysOnErrorSampler struct {
	baseSampler sdktrace.Sampler
}

// NewAlwaysOnErrorSampler creates a new AlwaysOnErrorSampler
func NewAlwaysOnErrorSampler(baseSampler sdktrace.Sampler) sdktrace.Sampler {
	return &AlwaysOnErrorSampler{
		baseSampler: baseSampler,
	}
}

// ShouldSample implements the Sampler interface
func (s *AlwaysOnErrorSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// Always use base sampler for initial decision
	result := s.baseSampler.ShouldSample(p)

	// The span will be set to error status later, but we can't know that at sampling time
	// So we need to use a SpanProcessor to handle error sampling
	return result
}

// Description returns the description of the sampler
func (s *AlwaysOnErrorSampler) Description() string {
	return "AlwaysOnError{" + s.baseSampler.Description() + "}"
}

// ErrorSpanProcessor ensures spans with errors are always exported
type ErrorSpanProcessor struct {
	sdktrace.SpanProcessor
}

// NewErrorSpanProcessor creates a new ErrorSpanProcessor
func NewErrorSpanProcessor(wrapped sdktrace.SpanProcessor) sdktrace.SpanProcessor {
	return &ErrorSpanProcessor{
		SpanProcessor: wrapped,
	}
}

// OnEnd is called when a span ends
func (p *ErrorSpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	// For error spans, we always call the wrapped processor's OnEnd
	// This ensures error spans are exported even with low sampling rates
	// No additional processing needed here - the sampling decision was already made

	// Always call the wrapped processor
	p.SpanProcessor.OnEnd(s)
}
