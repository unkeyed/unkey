package lazy

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestCounterRegisterPublishesZero(t *testing.T) {
	SetRegistry(prometheus.NewRegistry())

	counter := NewCounter(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "lazy",
		Name:      "register_probe_total",
		Help:      "Probe counter for the Register test.",
	})

	reg := registry.Load()
	require.NotNil(t, reg)

	before, err := reg.Gather()
	require.NoError(t, err)
	require.Empty(t, before)

	counter.Register()

	after, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, family := range after {
		if family.GetName() != "unkey_lazy_register_probe_total" {
			continue
		}
		found = true
		require.Len(t, family.GetMetric(), 1)
		require.Equal(t, float64(0), family.GetMetric()[0].GetCounter().GetValue())
	}
	require.True(t, found, "Register did not publish unkey_lazy_register_probe_total")
}
