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
	t.Cleanup(reset)
}

func TestSetRegistryMovesMetricsUsedBeforeIt(t *testing.T) {
	resetState(t)

	early := NewCounterVec(prometheus.CounterOpts{Name: "early_total", Help: "h"}, []string{"result"})
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

func TestBindCreatesSeriesOnFirstUseOnly(t *testing.T) {
	resetState(t)
	reg := prometheus.NewRegistry()
	SetRegistry(reg)

	lvs := []string{"hit"}
	child := NewCounterVec(prometheus.CounterOpts{Name: "bound_total", Help: "h"}, []string{"result"}).Bind(lvs...)
	lvs[0] = "changed"
	require.Equal(t, 0, series(t, reg, "bound_total"))

	child.Inc()
	child.Inc()

	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	metric := families[0].GetMetric()
	require.Len(t, metric, 1)
	require.Equal(t, "hit", metric[0].GetLabel()[0].GetValue())
	require.InDelta(t, 2, metric[0].GetCounter().GetValue(), 0)
}
