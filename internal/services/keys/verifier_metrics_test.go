package keys

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var metricsRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(metricsRegistry)
	os.Exit(m.Run())
}

const verificationsMetric = "unkey_key_verifications_total"

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

func TestKeyVerifier_VerifyRecordsTerminalStatus(t *testing.T) {
	t.Run("passed verification", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusValid)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusValid}
		require.NoError(t, k.Verify(t.Context()))

		require.Equal(t, 1.0, counterValue(t, verificationsMetric, labels)-before)
	})

	t.Run("rejected during verification", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusNotFound)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{
			Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
			Status: StatusValid,
		}
		require.NoError(t, k.Verify(t.Context(), WithKeyspaces("ks_other")))

		require.Equal(t, 1.0, counterValue(t, verificationsMetric, labels)-before)
	})
}

func TestKeyVerifier_VerifyRecordsLateRejections(t *testing.T) {
	for _, status := range []KeyStatus{
		StatusForbidden,
		StatusInsufficientPermissions,
		StatusRateLimited,
		StatusUsageExceeded,
	} {
		t.Run(string(status), func(t *testing.T) {
			labels := map[string]string{"type": "key", "code": string(status)}
			before := counterValue(t, verificationsMetric, labels)

			k := &KeyVerifier{Status: status}
			k.recordVerifyOutcome(StatusValid)

			require.Equal(t, 1.0, counterValue(t, verificationsMetric, labels)-before)
		})
	}
}

func TestKeyVerifier_VerifyLeavesStatusesGetOwns(t *testing.T) {
	t.Run("root key is recorded by Get", func(t *testing.T) {
		labels := map[string]string{"type": "root_key", "code": string(StatusValid)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusValid, isRootKey: true}
		k.recordVerifyOutcome(StatusValid)

		require.Equal(t, 0.0, counterValue(t, verificationsMetric, labels)-before)
	})

	t.Run("status already decided at Get", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusExpired)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusExpired}
		k.recordVerifyOutcome(StatusExpired)

		require.Equal(t, 0.0, counterValue(t, verificationsMetric, labels)-before)
	})
}
