package lazy

import (
	"fmt"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func resetState(t *testing.T) {
	t.Helper()
	reset := func() {
		state.mu.Lock()
		defer state.mu.Unlock()
		state.registry = nil
		state.explicit = false
		state.collectors = nil
	}
	reset()
	declarations.mu.Lock()
	declarations.primes = nil
	declarations.mu.Unlock()
	t.Cleanup(reset)
}

func TestSetRegistryMovesMetricsUsedBeforeIt(t *testing.T) {
	resetState(t)

	early := NewCounterVec(prometheus.CounterOpts{Name: "early_total", Help: "h"}, Values("result", "error"))
	early.WithLabelValues("error").Inc()

	reg := prometheus.NewRegistry()
	SetRegistry(reg)
	early.WithLabelValues("error").Inc()

	require.InDelta(t, 2, counterValue(t, reg, "early_total"), 0)
}

func TestSetRegistryFirstCallWins(t *testing.T) {
	resetState(t)

	first := prometheus.NewRegistry()
	second := prometheus.NewRegistry()
	SetRegistry(first)
	SetRegistry(second)

	c := NewCounter(prometheus.CounterOpts{Name: "after_total", Help: "h"})
	c.Inc()

	require.Equal(t, 1, series(t, first, "after_total"))
	require.Equal(t, 0, series(t, second, "after_total"))
}

func TestSetRegistryDuringConcurrentFirstUseLosesNoMetric(t *testing.T) {
	resetState(t)

	const n = 200
	counters := make([]*Counter, n)
	for i := range counters {
		counters[i] = NewCounter(prometheus.CounterOpts{Name: fmt.Sprintf("race_%d_total", i), Help: "h"})
	}

	reg := prometheus.NewRegistry()
	var wg sync.WaitGroup
	for _, c := range counters {
		wg.Go(c.Inc)
	}
	wg.Go(func() { SetRegistry(reg) })
	wg.Wait()

	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, n)
}

func TestSetRegistryPanicsOnConflict(t *testing.T) {
	resetState(t)

	NewCounter(prometheus.CounterOpts{Name: "dup_total", Help: "h"}).Inc()

	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewCounter(prometheus.CounterOpts{Name: "dup_total", Help: "h"}))
	require.Panics(t, func() { SetRegistry(reg) })
}

// series returns the number of series reg exports for the metric family name.
func series(t *testing.T, reg *prometheus.Registry, name string) int {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() == name {
			return len(f.GetMetric())
		}
	}
	return 0
}

// counterValue returns the value of the single series of the counter family name.
func counterValue(t *testing.T, reg *prometheus.Registry, name string) float64 {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() == name {
			require.Len(t, f.GetMetric(), 1)
			return f.GetMetric()[0].GetCounter().GetValue()
		}
	}
	require.Failf(t, "metric not found", "%s", name)
	return 0
}

func TestSetRegistryCreatesDeclaredSeriesAtZero(t *testing.T) {
	resetState(t)

	NewCounterVec(prometheus.CounterOpts{Name: "fixed_total", Help: "h"},
		Values("op", "read", "write"), Values("result", "success", "error"))
	NewCounterVec(prometheus.CounterOpts{Name: "dynamic_total", Help: "h"},
		Dynamic("resource"), Values("result", "success", "error"))
	NewHistogramVec(prometheus.HistogramOpts{Name: "fixed_seconds", Help: "h"}, Values("op", "read"))
	NewCounter(prometheus.CounterOpts{Name: "plain_total", Help: "h"})
	NewHistogram(prometheus.HistogramOpts{Name: "plain_seconds", Help: "h"})
	NewGauge(prometheus.GaugeOpts{Name: "plain_gauge", Help: "h"})

	reg := prometheus.NewRegistry()
	SetRegistry(reg)

	require.Equal(t, 4, series(t, reg, "fixed_total"))
	require.Equal(t, 0, series(t, reg, "dynamic_total"))
	require.Equal(t, 1, series(t, reg, "fixed_seconds"))
	require.Equal(t, 1, series(t, reg, "plain_total"))
	require.Equal(t, 1, series(t, reg, "plain_seconds"))
	require.Equal(t, 0, series(t, reg, "plain_gauge"))
	require.InDelta(t, 0, counterValue(t, reg, "plain_total"), 0)
}

func TestBindCounterCreatesSeriesAtZero(t *testing.T) {
	resetState(t)
	reg := prometheus.NewRegistry()
	SetRegistry(reg)

	vec := NewCounterVec(prometheus.CounterOpts{Name: "bound_total", Help: "h"}, Dynamic("resource"))
	child := vec.Bind("keys")
	require.Equal(t, 1, series(t, reg, "bound_total"))
	require.InDelta(t, 0, counterValue(t, reg, "bound_total"), 0)

	child.Inc()
	require.InDelta(t, 1, counterValue(t, reg, "bound_total"), 0)
}

func TestBindHistogramCreatesSeriesOnFirstUseOnly(t *testing.T) {
	resetState(t)
	reg := prometheus.NewRegistry()
	SetRegistry(reg)

	lvs := []string{"keys"}
	child := NewHistogramVec(prometheus.HistogramOpts{Name: "bound_seconds", Help: "h"}, Dynamic("resource")).Bind(lvs...)
	lvs[0] = "changed"
	require.Equal(t, 0, series(t, reg, "bound_seconds"))

	child.Observe(1)
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Equal(t, "keys", families[0].GetMetric()[0].GetLabel()[0].GetValue())
}
