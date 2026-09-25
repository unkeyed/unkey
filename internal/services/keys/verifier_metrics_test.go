package keys

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

const rejectionsMetric = "unkey_key_verification_rejections_total"

func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := metricsRegistry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		for _, m := range family.GetMetric() {
			got := make(map[string]string, len(m.GetLabel()))
			for _, pair := range m.GetLabel() {
				got[pair.GetName()] = pair.GetValue()
			}

			if len(got) != len(labels) {
				continue
			}

			match := true
			for k, v := range labels {
				if got[k] != v {
					match = false
					break
				}
			}

			if match {
				return m.GetCounter().GetValue()
			}
		}
	}

	return 0
}

func TestKeyVerifier_RecordsLateRejections(t *testing.T) {
	for _, status := range []KeyStatus{
		StatusForbidden,
		StatusInsufficientPermissions,
		StatusRateLimited,
		StatusUsageExceeded,
	} {
		t.Run(string(status), func(t *testing.T) {
			labels := map[string]string{"type": "key", "code": string(status)}
			before := counterValue(t, rejectionsMetric, labels)

			k := &KeyVerifier{Status: status}
			k.recordRejection(StatusValid)

			require.Equal(t, 1.0, counterValue(t, rejectionsMetric, labels)-before)
		})
	}
}

func TestKeyVerifier_RecordsRootKeyType(t *testing.T) {
	labels := map[string]string{"type": "root_key", "code": string(StatusRateLimited)}
	before := counterValue(t, rejectionsMetric, labels)

	k := &KeyVerifier{Status: StatusRateLimited, isRootKey: true}
	k.recordRejection(StatusValid)

	require.Equal(t, 1.0, counterValue(t, rejectionsMetric, labels)-before)
}

func TestKeyVerifier_IgnoresGetTimeStatuses(t *testing.T) {
	for _, status := range []KeyStatus{
		StatusValid,
		StatusNotFound,
		StatusDisabled,
		StatusExpired,
		StatusWorkspaceDisabled,
		StatusWorkspaceNotFound,
	} {
		t.Run(string(status), func(t *testing.T) {
			labels := map[string]string{"type": "key", "code": string(status)}
			before := counterValue(t, rejectionsMetric, labels)

			k := &KeyVerifier{Status: status}
			k.recordRejection(StatusValid)

			require.Equal(t, 0.0, counterValue(t, rejectionsMetric, labels)-before)
		})
	}
}

func TestKeyVerifier_IgnoresRejectionOnAlreadyInvalidKey(t *testing.T) {
	labels := map[string]string{"type": "key", "code": string(StatusRateLimited)}
	before := counterValue(t, rejectionsMetric, labels)

	k := &KeyVerifier{Status: StatusRateLimited}
	k.recordRejection(StatusExpired)

	require.Equal(t, 0.0, counterValue(t, rejectionsMetric, labels)-before)
}
