package keys

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
)

var metricsRegistry = prometheus.DefaultGatherer

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
		labels := map[string]string{"type": "key", "code": string(StatusValid), "reason": string(ReasonNone)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusValid, reason: ReasonNone}
		require.NoError(t, k.Verify(t.Context()))

		require.Equal(t, 1.0, counterValue(t, verificationsMetric, labels)-before)
	})

	t.Run("rejected during verification", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusNotFound), "reason": string(ReasonKeyspaceMismatch)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{
			Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
			Status: StatusValid,
			reason: ReasonNone,
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
			labels := map[string]string{"type": "key", "code": string(status), "reason": string(ReasonRateLimited)}
			before := counterValue(t, verificationsMetric, labels)

			k := &KeyVerifier{Status: status, reason: ReasonRateLimited}
			k.recordVerifyOutcome(StatusValid)

			require.Equal(t, 1.0, counterValue(t, verificationsMetric, labels)-before)
		})
	}
}

func TestKeyVerifier_VerifyLeavesStatusesGetOwns(t *testing.T) {
	t.Run("root key is recorded by Get", func(t *testing.T) {
		labels := map[string]string{"type": "root_key", "code": string(StatusValid), "reason": string(ReasonNone)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusValid, reason: ReasonNone, isRootKey: true}
		k.recordVerifyOutcome(StatusValid)

		require.Equal(t, 0.0, counterValue(t, verificationsMetric, labels)-before)
	})

	t.Run("status already decided at Get", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusExpired), "reason": string(ReasonExpired)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusExpired, reason: ReasonExpired}
		k.recordVerifyOutcome(StatusExpired)

		require.Equal(t, 0.0, counterValue(t, verificationsMetric, labels)-before)
	})
}

func TestKeyVerifier_HideRecordsUndecidedKeysOnce(t *testing.T) {
	t.Run("key that passed Get", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusNotFound), "reason": string(ReasonWorkspaceMismatch)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusValid, reason: ReasonNone}
		k.Hide(ReasonWorkspaceMismatch)

		require.Equal(t, StatusNotFound, k.Status)
		require.Equal(t, 1.0, counterValue(t, verificationsMetric, labels)-before)
	})

	t.Run("key Get already rejected", func(t *testing.T) {
		labels := map[string]string{"type": "key", "code": string(StatusNotFound), "reason": string(ReasonPermissionDenied)}
		before := counterValue(t, verificationsMetric, labels)

		k := &KeyVerifier{Status: StatusDisabled, reason: ReasonDisabled}
		k.Hide(ReasonPermissionDenied)

		require.Equal(t, 0.0, counterValue(t, verificationsMetric, labels)-before)
	})
}
