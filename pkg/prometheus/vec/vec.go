// Package vec creates Prometheus metric vectors that export every known label
// combination at 0 from process start.
//
// A vector without children exports nothing, so dashboards and alerts cannot
// tell "no errors" from "metric missing", and rate() misses the first
// increment. Declaring the values a label can take next to the label name lets
// the constructor create those children at registration time. Vectors are
// registered with [promauto] in the default registry.
//
//	var Operations = vec.NewCounter(prometheus.CounterOpts{
//		Namespace: "unkey",
//		Name:      "vault_operations_total",
//		Help:      "Vault operations by operation and outcome.",
//	},
//		vec.Values("operation", "encrypt", "decrypt"),
//		vec.Values("outcome", "success", "error"),
//	)
//
// Use [Dynamic] for labels whose values are only known at run time, such as
// IDs or names. A vector with a dynamic label is not primed, because the
// combinations are unknown.
//
// The linter forbids promauto vector constructors outside this package.
package vec

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Label declares one label of a vector. Create it with [Values] or [Dynamic].
type Label struct {
	name   string
	values []string
}

// Values declares a label with the complete set of values the code emits.
func Values(name string, values ...string) Label {
	if len(values) == 0 {
		panic(fmt.Sprintf("vec: label %q has no values; use vec.Dynamic", name))
	}
	return Label{name: name, values: values}
}

// Dynamic declares a label whose values are only known at run time.
func Dynamic(name string) Label {
	return Label{name: name, values: nil}
}

// NewCounter registers a counter vector and creates every known label
// combination at 0.
func NewCounter(opts prometheus.CounterOpts, labels ...Label) *prometheus.CounterVec {
	v := promauto.NewCounterVec(opts, names(labels))
	prime(labels, func(lvs []string) { v.WithLabelValues(lvs...) })
	return v
}

// NewHistogram registers a histogram vector and creates every known label
// combination with no observations.
func NewHistogram(opts prometheus.HistogramOpts, labels ...Label) *prometheus.HistogramVec {
	v := promauto.NewHistogramVec(opts, names(labels))
	prime(labels, func(lvs []string) { v.WithLabelValues(lvs...) })
	return v
}

// NewGauge registers a gauge vector and creates every known label combination
// at 0.
func NewGauge(opts prometheus.GaugeOpts, labels ...Label) *prometheus.GaugeVec {
	v := promauto.NewGaugeVec(opts, names(labels))
	prime(labels, func(lvs []string) { v.WithLabelValues(lvs...) })
	return v
}

func names(labels []Label) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.name
	}
	return out
}

func prime(labels []Label, create func([]string)) {
	for _, l := range labels {
		if l.values == nil {
			return
		}
	}
	lvs := make([]string, len(labels))
	var walk func(int)
	walk = func(i int) {
		if i == len(labels) {
			create(lvs)
			return
		}
		for _, v := range labels[i].values {
			lvs[i] = v
			walk(i + 1)
		}
	}
	walk(0)
}
