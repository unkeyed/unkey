package clickhouseuser_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestReconcileUser_Integration(t *testing.T) {
	h := harness.New(t)
	workspace := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{
		RequestsPerMonth:  1_000_000,
		LogsRetentionDays: 30,
	})

	const maxQueriesPerWindow = int32(2_000)
	userClient := hydrav1.NewClickhouseUserServiceIngressClient(h.Restate, workspace.ID)
	_, err := userClient.ConfigureUser().Request(h.Ctx, &hydrav1.ConfigureUserRequest{
		MaxQueriesPerWindow: ptr.P(maxQueriesPerWindow),
	})
	require.NoError(t, err)

	settings, err := h.DB.FindClickhouseWorkspaceSettingsByWorkspaceID(h.Ctx, workspace.ID)
	require.NoError(t, err)
	password, err := h.VaultClient.Decrypt(h.Ctx, &vaultv1.DecryptRequest{
		Keyring:   workspace.ID,
		Encrypted: settings.ClickhousePasswordEncrypted,
	})
	require.NoError(t, err)

	options, err := ch.ParseDSN(h.ClickHouseDSN)
	require.NoError(t, err)
	options.Auth.Username = workspace.ID
	options.Auth.Password = password.GetPlaintext()
	workspaceConn, err := ch.Open(options)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workspaceConn.Close()) })

	// Simulate a user provisioned by an older control-worker release.
	err = h.ClickHouse.Exec(h.Ctx, fmt.Sprintf(
		"REVOKE SELECT(path) ON default.frontline_requests_raw_v1 FROM %s",
		workspace.ID,
	))
	require.NoError(t, err)
	var count uint64
	queryCtx, cancelQuery := context.WithTimeout(t.Context(), 10*time.Second)
	err = workspaceConn.QueryRow(queryCtx,
		"SELECT countIf(path = '') FROM default.frontline_requests_raw_v1",
	).Scan(&count)
	cancelQuery()
	require.Error(t, err, "the stale user should not be able to read the revoked column")

	_, err = userClient.ReconcileUser().Request(
		h.Ctx,
		&hydrav1.ReconcileUserRequest{},
	)
	require.NoError(t, err)

	queryCtx, cancelQuery = context.WithTimeout(t.Context(), 10*time.Second)
	err = workspaceConn.QueryRow(queryCtx,
		"SELECT countIf(path = '') FROM default.frontline_requests_raw_v1",
	).Scan(&count)
	cancelQuery()
	require.NoError(t, err, "reconciliation should restore the current gateway request grant")

	settings, err = h.DB.FindClickhouseWorkspaceSettingsByWorkspaceID(h.Ctx, workspace.ID)
	require.NoError(t, err)
	require.Equal(t, maxQueriesPerWindow, settings.ClickhouseMaxQueriesPerWindow,
		"reconciliation must preserve stored workspace limits")
}
