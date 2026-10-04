package vercel

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestCollectorsExportUnlabeledHealthMetrics(t *testing.T) {
	p, err := New(Config{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0})
	require.NoError(t, err)
	reg := prometheus.NewRegistry()
	for _, collector := range p.Collectors() {
		require.NoError(t, reg.Register(collector))
	}

	families, err := reg.Gather()
	require.NoError(t, err)
	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
		for _, metric := range family.GetMetric() {
			require.Empty(t, metric.GetLabel(), "%s carries labels", family.GetName())
		}
	}
	require.ElementsMatch(t, []string{
		"unkey_feature_flags_ready",
		"unkey_feature_flags_snapshot_age_seconds",
		"unkey_feature_flags_refresh_failures_total",
	}, names)
}
