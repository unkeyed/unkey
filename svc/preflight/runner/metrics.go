package runner

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// Prometheus metrics emitted by the Runner. Declared at package
// scope with promauto so they register against the default registry
// at init() and are never emitted from probe code.
var (
	runTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "preflight_run_total",
		Help: "Number of preflight probe runs by outcome.",
	}, []string{"suite", "probe", "result", "region"})

	probeDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "preflight_probe_duration_seconds",
		Help:    "Wall-clock duration of a probe from start to finish.",
		Buckets: prometheus.ExponentialBucketsRange(0.05, 300, 12),
	}, []string{"suite", "probe", "region"})

	phaseDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "preflight_phase_duration_seconds",
		Help:    "Wall-clock duration of a named phase inside a probe.",
		Buckets: prometheus.ExponentialBucketsRange(0.05, 300, 12),
	}, []string{"suite", "probe", "phase", "region"})
)

// resultLabel returns the value for the "result" label on
// preflight_run_total, collapsing shadow-mode failures into a separate
// bucket so PrometheusRules can filter them out.
func (r *Runner) resultLabel(probeName string, ok bool) string {
	if ok {
		return "success"
	}
	if r.shadow[probeName] {
		return "shadow_fail"
	}
	return "fail"
}

func (r *Runner) emitMetrics(probeName string, res core.Result, dur time.Duration) {
	runTotal.WithLabelValues(r.suite, probeName, r.resultLabel(probeName, res.OK), r.region).Inc()
	probeDuration.WithLabelValues(r.suite, probeName, r.region).Observe(dur.Seconds())
	for _, p := range res.Phases {
		phaseDuration.WithLabelValues(r.suite, probeName, p.Name, r.region).Observe(p.Duration.Seconds())
	}
}

func (r *Runner) logResult(probeName string, res core.Result, dur time.Duration) {
	attrs := []any{
		"suite", r.suite,
		"probe", probeName,
		"region", r.region,
		"duration_ms", dur.Milliseconds(),
		"ok", res.OK,
	}
	for k, v := range res.Dims {
		attrs = append(attrs, k, v)
	}
	if res.Err != nil {
		attrs = append(attrs, "error", res.Err.Error())
	}

	switch {
	case res.OK:
		logger.Info("preflight probe ok", attrs...)
	case r.shadow[probeName]:
		logger.Warn("preflight probe failed (shadow)", attrs...)
	default:
		logger.Error("preflight probe failed", attrs...)
	}
}
