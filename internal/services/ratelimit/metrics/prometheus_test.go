package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

func TestRegisterZeroBaselinePublishesFailureCounters(t *testing.T) {
	reg := prometheus.NewRegistry()
	lazy.SetRegistry(reg)

	before, err := reg.Gather()
	require.NoError(t, err)
	require.Empty(t, before)

	RegisterZeroBaseline()

	after, err := reg.Gather()
	require.NoError(t, err)

	values := map[string]float64{}
	for _, family := range after {
		for _, m := range family.GetMetric() {
			values[family.GetName()] = m.GetCounter().GetValue()
		}
	}

	require.Equal(t, map[string]float64{
		"unkey_ratelimit_cas_exhausted_total":      0,
		"unkey_ratelimit_global_pull_errors_total": 0,
		"unkey_ratelimit_global_push_errors_total": 0,
	}, values)
}
