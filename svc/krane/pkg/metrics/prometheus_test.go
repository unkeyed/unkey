package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestSeriesExportedBeforeFirstUse(t *testing.T) {
	reg := prometheus.DefaultGatherer

	seriesWant := map[string]int{
		"unkey_krane_watcher_full_syncs_total":                1,
		"unkey_krane_stream_events_received_total":            1,
		"unkey_krane_stream_errors_total":                     1,
		"unkey_krane_full_sync_errors_total":                  2,
		"unkey_krane_full_sync_events_received_total":         1,
		"unkey_krane_full_sync_duration_seconds":              1,
		"unkey_krane_instance_events_report_failures_total":   1,
		"unkey_krane_last_successful_checkpoint_unix_seconds": 1,
		"unkey_krane_leader":                                  1,
		"unkey_krane_stream_connections_total":                2,
		"unkey_krane_report_status_duration_seconds":          2,
		"unkey_krane_dispatch_total":                          4,
		"unkey_krane_reconcile_operations_total":              4,
		"unkey_krane_rollout_gate_workloads":                  3,
		"unkey_krane_pod_watch_reconnects_total":              2,
		"unkey_krane_instance_events_dedup_dropped_total":     3,
		"unkey_krane_resync_corrections_total":                1,
		"unkey_krane_report_deduped_total":                    1,
		"unkey_krane_pod_watch_delivery_lag_seconds":          1,
	}
	families := gather(t, reg)
	for name, want := range seriesWant {
		require.Contains(t, families, name)
		require.Len(t, families[name].GetMetric(), want, name)
	}

	Leader.Set(1)
	RolloutGateWorkloads.WithLabelValues("rolling").Set(3)
	LastSuccessfulCheckpointUnixSeconds.Set(1)

	ResetLeaderGauges()

	families = gather(t, reg)
	for _, name := range []string{
		"unkey_krane_leader",
		"unkey_krane_last_successful_checkpoint_unix_seconds",
		"unkey_krane_rollout_gate_workloads",
	} {
		for _, m := range families[name].GetMetric() {
			require.Zero(t, m.GetGauge().GetValue(), name)
		}
	}
}

func gather(t *testing.T, reg prometheus.Gatherer) map[string]*dto.MetricFamily {
	t.Helper()

	families, err := reg.Gather()
	require.NoError(t, err)

	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		byName[family.GetName()] = family
	}
	return byName
}
