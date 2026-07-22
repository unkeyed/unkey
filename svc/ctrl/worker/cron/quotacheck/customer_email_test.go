package quotacheck

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestCustomerEmailTemplate(t *testing.T) {
	tests := []struct {
		name       string
		isFreeTier bool
		attempt    int64
		template   string
		ok         bool
	}{
		{name: "first free-tier email", isFreeTier: true, attempt: 0, template: usageExceededTemplate, ok: true},
		{name: "second free-tier email", isFreeTier: true, attempt: 1, template: usageRatelimitFollowUpTemplate, ok: true},
		{name: "no third email", isFreeTier: true, attempt: 2, template: "", ok: false},
		{name: "paid tier skipped", isFreeTier: false, attempt: 0, template: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, ok := customerEmailTemplate(tt.isFreeTier, tt.attempt)
			require.Equal(t, tt.template, template)
			require.Equal(t, tt.ok, ok)
		})
	}
}

func TestCustomerEmailIdempotencyKey(t *testing.T) {
	require.Equal(t, "quota-alert/ws_123/2026-07/2", customerEmailIdempotencyKey("ws_123", "2026-07", 1))
}

func TestSendCustomerEmail_DisabledDoesNotConsumeAttempt(t *testing.T) {
	h := &Handler{customerEmailEnabled: false}
	exceeded := exceededWorkspace{Workspace: db.GetWorkspacesForQuotaCheckByIDsRow{
		StripeSubscriptionID: sql.NullString{Valid: false}, // no subscription -> Free tier
	}}

	for range 2 {
		sent, err := h.sendCustomerEmail(nil, "2026-07", 2026, 0, exceeded)
		require.NoError(t, err)
		require.False(t, sent)
	}

	template, ok := customerEmailTemplate(true, 0)
	require.True(t, ok)
	require.Equal(t, usageExceededTemplate, template)
}
