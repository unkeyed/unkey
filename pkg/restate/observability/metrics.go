package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// Buckets covers DB-call-fast (~100ms) through long-running deploys
// (~hour). Tuned for workflow steps; not a general-purpose histogram.
var workflowBuckets = []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}

var (
	// stepDuration tracks per-attempt duration of an instrumented restate.Run
	// step. Each retry contributes a sample; aggregate via PromQL to derive
	// total wall-clock per step if needed.
	stepDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "workflow_step_duration_seconds",
			Help:      "Duration of an instrumented Restate workflow step (per attempt).",
			Buckets:   workflowBuckets,
		},
		[]string{"workflow", "step"},
	)

	// stepTotal counts step terminations by outcome and error category.
	//
	// Labels:
	//   - workflow: stable workflow name (e.g. "deploy", "sentinel_rollout")
	//   - step: stable step name within the workflow
	//   - outcome: "success" | "failed" | "cancelled"
	//   - error_category: "none" | "user" | "app" | "provider" | "infra" | "cancelled"
	stepTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "workflow_step_total",
			Help:      "Total instrumented Restate workflow step executions.",
		},
		[]string{"workflow", "step", "outcome", "error_category"},
	)

	// runDuration tracks duration of an entire workflow handler invocation.
	// Used by Run() for top-level handlers and single-shot cron jobs.
	runDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "workflow_run_duration_seconds",
			Help:      "Duration of an instrumented Restate workflow handler.",
			Buckets:   workflowBuckets,
		},
		[]string{"workflow", "outcome"},
	)

	// runTotal counts workflow handler invocations by outcome and category.
	runTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "workflow_run_total",
			Help:      "Total instrumented Restate workflow handler invocations.",
		},
		[]string{"workflow", "outcome", "error_category"},
	)
)
