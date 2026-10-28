package vec

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewCounterPrimesEveryCombination(t *testing.T) {
	v := NewCounter(prometheus.CounterOpts{Name: "vec_test_primed_total", Help: "test"},
		Values("operation", "get", "put", "list"),
		Values("outcome", "success", "error"),
	)

	require.Equal(t, 6, count(v))
	v.WithLabelValues("list", "error")
	require.Equal(t, 6, count(v))
}

func TestDynamicLabelSkipsPriming(t *testing.T) {
	v := NewHistogram(prometheus.HistogramOpts{Name: "vec_test_dynamic_seconds", Help: "test"},
		Dynamic("resource"),
		Values("outcome", "success", "error"),
	)

	require.Equal(t, 0, count(v))
}

func TestValuesWithoutValuesPanics(t *testing.T) {
	require.Panics(t, func() { Values("outcome") })
}

func count(c prometheus.Collector) int {
	ch := make(chan prometheus.Metric, 100)
	c.Collect(ch)
	close(ch)
	return len(ch)
}
