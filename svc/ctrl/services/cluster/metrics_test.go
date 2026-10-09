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

type catalogCounters struct {
	builds           float64
	bootstraps       float64
	refreshedCallers float64
}

func catalogWork(t *testing.T) catalogCounters {
	t.Helper()
	families, err := metricsRegistry.Gather()
	require.NoError(t, err)
	var work catalogCounters
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			value := metric.GetCounter().GetValue()
			kind := ""
			for _, label := range metric.GetLabel() {
				if label.GetName() == "kind" {
					kind = label.GetValue()
				}
			}
			switch {
			case family.GetName() == "unkey_control_private_network_catalog_builds_total":
				work.builds = value
			case family.GetName() == "unkey_control_private_network_catalog_refreshes_total" && kind == "bootstrap":
				work.bootstraps = value
			case family.GetName() == "unkey_control_private_network_catalog_refreshed_callers_total" && kind == "update":
				work.refreshedCallers = value
			}
		}
	}
	return work
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
