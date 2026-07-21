package analytics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestEnsureProvisioningReady_FailsClosed(t *testing.T) {
	for _, state := range []string{"", "pending", "failed"} {
		t.Run(state, func(t *testing.T) {
			settings := db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
				ClickhouseWorkspaceSetting: db.ClickhouseWorkspaceSetting{ProvisioningState: state},
			}

			// Security guarantee: API connection acquisition rejects every state except explicit readiness.
			require.Error(t, ensureProvisioningReady(settings))
		})
	}

	settings := db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseWorkspaceSetting: db.ClickhouseWorkspaceSetting{ProvisioningState: provisioningStateReady},
	}
	require.NoError(t, ensureProvisioningReady(settings))
}
