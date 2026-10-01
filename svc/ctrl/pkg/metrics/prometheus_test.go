package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

func TestEnvironmentDeletionAgeTracksOldestUntilCompletion(t *testing.T) {
	registry := prometheus.NewRegistry()
	lazy.SetRegistry(registry)
	t.Cleanup(func() { ObserveEnvironmentDeletion(0) })

	now := time.Now()
	ObserveEnvironmentDeletion(now.Add(-20 * time.Minute).UnixMilli())
	require.InDelta(t, 20*time.Minute.Seconds(), deletionAge(t, registry), 1)

	ObserveEnvironmentDeletion(now.Add(-5 * time.Minute).UnixMilli())
	require.InDelta(t, 5*time.Minute.Seconds(), deletionAge(t, registry), 1)

	ObserveEnvironmentDeletion(0)
	require.Zero(t, deletionAge(t, registry))
	ObserveEnvironmentDeletion(now.Add(time.Minute).UnixMilli())
	require.Zero(t, deletionAge(t, registry))
}

func deletionAge(t *testing.T, registry *prometheus.Registry) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "unkey_control_environment_deletion_age_seconds" {
			require.Len(t, family.Metric, 1)
			return family.Metric[0].GetGauge().GetValue()
		}
	}
	t.Fatal("deletion age metric missing")
	return 0
}
