package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeleteRegionalSettingsRemovesOnlyUnreferencedPolicies(t *testing.T) {
	mysql := containers.MySQL(t)
	database, err := db.New(mysql.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	for _, concurrent := range []bool{false, true} {
		name := "sequential"
		if concurrent {
			name = "concurrent"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			workspaceID := uid.New(uid.WorkspacePrefix)
			envID := uid.New(uid.EnvironmentPrefix)
			otherEnvID := uid.New(uid.EnvironmentPrefix)
			privatePolicyID := uid.New("hap")
			sharedPolicyID := uid.New("hap")
			unattachedPolicyID := uid.New("hap")
			t.Cleanup(func() {
				_, err := database.RW().ExecContext(context.Background(), "DELETE FROM app_regional_settings WHERE workspace_id = ?", workspaceID)
				require.NoError(t, err)
				_, err = database.RW().ExecContext(context.Background(), "DELETE FROM horizontal_autoscaling_policies WHERE workspace_id = ?", workspaceID)
				require.NoError(t, err)
			})
			for _, id := range []string{privatePolicyID, sharedPolicyID, unattachedPolicyID} {
				_, err := database.RW().ExecContext(ctx, "INSERT INTO horizontal_autoscaling_policies (id, workspace_id, replicas_min, replicas_max, created_at) VALUES (?, ?, 1, 3, 1)", id, workspaceID)
				require.NoError(t, err)
			}
			for _, setting := range []struct{ envID, policyID string }{
				{envID, privatePolicyID}, {envID, privatePolicyID},
				{envID, sharedPolicyID}, {otherEnvID, sharedPolicyID},
			} {
				_, err := database.RW().ExecContext(ctx, "INSERT INTO app_regional_settings (workspace_id, app_id, environment_id, region_id, horizontal_autoscaling_policy_id, created_at) VALUES (?, ?, ?, ?, ?, 1)", workspaceID, workspaceID, setting.envID, uid.New(uid.RegionPrefix), setting.policyID)
				require.NoError(t, err)
			}
			count := func(table, id string) int {
				t.Helper()
				var n int
				query := "SELECT COUNT(*) FROM horizontal_autoscaling_policies WHERE id = ?"
				if table == "settings" {
					query = "SELECT COUNT(*) FROM app_regional_settings WHERE environment_id = ?"
				}
				require.NoError(t, database.RW().QueryRowContext(ctx, query, id).Scan(&n))
				return n
			}
			remove := func(id string) error {
				return db.TxRetry(ctx, database.RW(), func(txCtx context.Context, tx db.DBTX) error {
					return db.NewQueries(tx).DeleteAppRegionalSettingsByEnvironmentId(txCtx, id)
				})
			}
			if concurrent {
				start := make(chan struct{})
				done := make(chan error, 2)
				for _, id := range []string{envID, otherEnvID} {
					go func() {
						<-start
						done <- remove(id)
					}()
				}
				close(start)
				require.NoError(t, <-done)
				require.NoError(t, <-done)
			} else {
				require.NoError(t, remove(envID))
				require.Equal(t, 0, count("policy", privatePolicyID))
				require.Equal(t, 1, count("policy", sharedPolicyID))
				require.Equal(t, 1, count("settings", otherEnvID))
				require.NoError(t, remove(otherEnvID))
			}
			require.Equal(t, 0, count("settings", envID))
			require.Equal(t, 0, count("settings", otherEnvID))
			require.Equal(t, 0, count("policy", privatePolicyID))
			require.Equal(t, 0, count("policy", sharedPolicyID))
			require.Equal(t, 1, count("policy", unattachedPolicyID))
			require.NoError(t, remove(envID))
		})
	}
}
