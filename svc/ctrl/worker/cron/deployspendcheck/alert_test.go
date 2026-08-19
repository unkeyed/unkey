package deployspendcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/billingperiod"
)

func TestBudgetAlertIdempotencyKey(t *testing.T) {
	require.Equal(t, "budget-alert/ws_abc/2026-06/10000/75", budgetAlertIdempotencyKey("ws_abc", "2026-06", 10000, 75))
	// A different budget yields a different key, so a threshold re-crossed after a
	// budget change is not deduped against the old budget's send.
	require.NotEqual(t,
		budgetAlertIdempotencyKey("ws_abc", "2026-06", 10000, 75),
		budgetAlertIdempotencyKey("ws_abc", "2026-06", 20000, 75),
	)
}

// TestPreviousPeriodStateKeys locks the key derivation: the check clears the
// previous period's period-scoped VO state keys on each in-period tick so an
// alerted or suspended workspace does not accrue an immortal key pair per month.
// The keys cleared must be exactly the prior period's, including across a year
// boundary. (The full month-rollover behavior is not exercised end to end
// because the integration harness runs on a real container clock, so a January
// tick cannot synthesize December state; this pins the derivation the clear
// relies on.)
func TestPreviousPeriodStateKeys(t *testing.T) {
	jul, err := billingperiod.Parse("2026-07")
	require.NoError(t, err)
	require.Equal(t, "spend_alert_high_water:2026-06", alertHighWaterKey(jul.Prev().Key()))
	require.Equal(t, "spend_suspend_generation:2026-06", suspendGenerationKey(jul.Prev().Key()))

	jan, err := billingperiod.Parse("2026-01")
	require.NoError(t, err)
	require.Equal(t, "spend_alert_high_water:2025-12", alertHighWaterKey(jan.Prev().Key()))
	require.Equal(t, "spend_suspend_generation:2025-12", suspendGenerationKey(jan.Prev().Key()))
}

func TestStoppedAlertIdempotencyKey(t *testing.T) {
	require.Equal(t, "budget-stopped/ws_abc/2026-06/1", stoppedAlertIdempotencyKey("ws_abc", "2026-06", 1))
	// A later suspension (higher generation) is a distinct send, not a dedup of
	// the first suspension.
	require.NotEqual(t,
		stoppedAlertIdempotencyKey("ws_abc", "2026-06", 1),
		stoppedAlertIdempotencyKey("ws_abc", "2026-06", 2),
	)
}
