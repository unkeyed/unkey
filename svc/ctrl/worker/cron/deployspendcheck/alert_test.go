package deployspendcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBudgetAlertIdempotencyKey(t *testing.T) {
	require.Equal(t, "budget-alert/ws_abc/2026-06/75", budgetAlertIdempotencyKey("ws_abc", "2026-06", 75))
}
