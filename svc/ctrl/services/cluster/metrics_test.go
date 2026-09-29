package cluster

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var metricsRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(metricsRegistry)
	os.Exit(m.Run())
}

func snapshotOutcomes(t *testing.T) map[string]float64 {
	t.Helper()
	families, err := metricsRegistry.Gather()
	require.NoError(t, err)
	outcomes := map[string]float64{}
	for _, family := range families {
		if family.GetName() != "unkey_control_private_network_snapshots_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "result" {
					outcomes[label.GetValue()] = metric.GetCounter().GetValue()
				}
			}
		}
	}
	return outcomes
}

func snapshotReads(t *testing.T) uint64 {
	t.Helper()
	families, err := metricsRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "unkey_control_private_network_snapshot_read_duration_seconds" {
			return family.GetMetric()[0].GetHistogram().GetSampleCount()
		}
	}
	return 0
}

func outcomeDelta(before, after map[string]float64) map[string]float64 {
	changed := map[string]float64{}
	for result, value := range after {
		if value != before[result] {
			changed[result] = value - before[result]
		}
	}
	return changed
}
