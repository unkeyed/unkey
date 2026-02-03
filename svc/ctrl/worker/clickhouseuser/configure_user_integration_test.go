package clickhouseuser_test

import (
	"fmt"
	"testing"

	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestConfigureUser_Integration(t *testing.T) {
	h := harness.New(t)

	// Create ingress client for calling Restate services
	ingressClient := ingress.NewClient(h.RestateIngress)

	t.Run("creates new user with default settings", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{
			RequestsPerMonth:  1_000_000,
			LogsRetentionDays: 30,
		})

		client := hydrav1.NewClickhouseUserServiceIngressClient(ingressClient, ws.ID)
		_, err := client.ConfigureUser().Request(h.Ctx, &hydrav1.ConfigureUserRequest{})
		require.NoError(t, err)

		// Verify the user was created in MySQL
		settings, err := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(h.Ctx, h.DB.RO(), ws.ID)
		require.NoError(t, err)
		require.Equal(t, ws.ID, settings.ClickhouseWorkspaceSetting.WorkspaceID)
		require.Equal(t, ws.ID, settings.ClickhouseWorkspaceSetting.Username)
		require.NotEmpty(t, settings.ClickhouseWorkspaceSetting.PasswordEncrypted)

		// Verify the user exists in ClickHouse
		var userName string
		err = h.ClickHouseConn.QueryRow(h.Ctx, "SELECT name FROM system.users WHERE name = ?", ws.ID).Scan(&userName)
		require.NoError(t, err)
		require.Equal(t, ws.ID, userName)

		// Verify quota was created
		var quotaName string
		err = h.ClickHouseConn.QueryRow(h.Ctx,
			"SELECT name FROM system.quotas WHERE name = ?",
			fmt.Sprintf("workspace_%s_quota", ws.ID)).Scan(&quotaName)
		require.NoError(t, err)

		// Verify settings profile was created
		var profileName string
		err = h.ClickHouseConn.QueryRow(h.Ctx,
			"SELECT name FROM system.settings_profiles WHERE name = ?",
			fmt.Sprintf("workspace_%s_profile", ws.ID)).Scan(&profileName)
		require.NoError(t, err)
	})

	t.Run("updates existing user settings", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{
			RequestsPerMonth:  1_000_000,
			LogsRetentionDays: 30,
		})

		client := hydrav1.NewClickhouseUserServiceIngressClient(ingressClient, ws.ID)

		// Create user first time
		_, err := client.ConfigureUser().Request(h.Ctx, &hydrav1.ConfigureUserRequest{})
		require.NoError(t, err)

		// Get the initial settings
		initial, err := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(h.Ctx, h.DB.RO(), ws.ID)
		require.NoError(t, err)
		initialPassword := initial.ClickhouseWorkspaceSetting.PasswordEncrypted

		// Call ConfigureUser again with different settings
		_, err = client.ConfigureUser().Request(h.Ctx, &hydrav1.ConfigureUserRequest{
			MaxQueriesPerWindow: ptr.P[int32](2000),
		})
		require.NoError(t, err)

		// Verify password was preserved (not regenerated)
		updated, err := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(h.Ctx, h.DB.RO(), ws.ID)
		require.NoError(t, err)
		require.Equal(t, initialPassword, updated.ClickhouseWorkspaceSetting.PasswordEncrypted)

		// Verify the new settings were applied
		require.Equal(t, int32(2000), updated.ClickhouseWorkspaceSetting.MaxQueriesPerWindow)
	})
}
