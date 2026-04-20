package observability

import (
	"time"

	restate "github.com/restatedev/sdk-go"
)

// Step wraps restate.Run and emits per-attempt duration + outcome metrics.
//
// Metrics are emitted from inside the restate.Run callback so they only fire
// on actual execution, never on journal replay. Each retry contributes one
// duration sample and one stepTotal increment — aggregate via PromQL to derive
// total wall-clock per logical step if needed.
//
// Call sites should use stable, low-cardinality workflow and step names. High-
// cardinality context (deployment_id, domain, etc.) belongs in resource state
// or the Restate UI, not in Prometheus labels.
//
// If the caller does not pass restate.WithName(...), the step name is used so
// the Restate UI shows a sensible label.
func Step[T any](
	ctx restate.Context,
	workflow string,
	step string,
	fn func(ctx restate.RunContext) (T, error),
	opts ...restate.RunOption,
) (T, error) {
	opts = append([]restate.RunOption{restate.WithName(step)}, opts...)
	return restate.Run(ctx, func(rc restate.RunContext) (T, error) {
		start := time.Now()
		result, err := fn(rc)
		record(workflow, step, time.Since(start), err)
		return result, err
	}, opts...)
}

// StepVoid is the void-returning variant of Step for callbacks without a
// useful return value.
func StepVoid(
	ctx restate.Context,
	workflow string,
	step string,
	fn func(ctx restate.RunContext) error,
	opts ...restate.RunOption,
) error {
	opts = append([]restate.RunOption{restate.WithName(step)}, opts...)
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		start := time.Now()
		err := fn(rc)
		record(workflow, step, time.Since(start), err)
		return err
	}, opts...)
}

func record(workflow, step string, d time.Duration, err error) {
	outcome, category := Classify(err)
	stepDuration.WithLabelValues(workflow, step).Observe(d.Seconds())
	stepTotal.WithLabelValues(workflow, step, outcome, category).Inc()
}

// RecordPhase emits step metrics for a logical phase whose body is not a
// single restate.Run call. The caller has already classified the outcome
// (typically by passing the phase error through Classify). Use Step or
// StepVoid in preference; this exists for the deploy phase wrapper which
// composes multiple inner restate.Run calls.
func RecordPhase(workflow, step string, d time.Duration, outcome, category string) {
	stepDuration.WithLabelValues(workflow, step).Observe(d.Seconds())
	stepTotal.WithLabelValues(workflow, step, outcome, category).Inc()
}
