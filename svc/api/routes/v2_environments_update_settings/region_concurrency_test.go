package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_update_settings"
	"golang.org/x/sync/errgroup"
)

// TestUpdateSettingsConcurrentRegions verifies that LockEnvironmentForUpdate
// serializes concurrent region reconciliations for the same environment.
//
// Region updates "replace the set": each request reads the current regions then
// upserts the desired ones and deletes the rest. Without the lock, two requests
// reading before either commits can interleave into a merged set. With the lock
// they run one at a time, so the final state is always exactly one submitted set.
func TestUpdateSettingsConcurrentRegions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	seedRegions(t, h, "us-east-1", "eu-west-1")

	// Competing requests: half want only us-east-1, half want only eu-west-1.
	// Each is a full "replace the set" with a single, different region.
	numConcurrent := 10
	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			regions := []openapi.RegionSetting{regionSetting("us-east-1", 1, 3)}
			if i%2 == 1 {
				regions = []openapi.RegionSetting{regionSetting("eu-west-1", 1, 2)}
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project:     env.projectID,
				App:         env.appID,
				Environment: env.environmentID,
				Regions:     &regions,
			})
			if res.Status != 200 {
				return fmt.Errorf("request %d: unexpected status %d: %s", i, res.Status, res.RawBody)
			}
			return nil
		})
	}
	require.NoError(t, g.Wait(), "all concurrent region updates should succeed without deadlock")

	// Exactly one region survives (last writer wins, never a merged set).
	rows, err := db.Query.ListAppRegionalSettingsByAppEnv(ctx, h.DB.RO(), db.ListAppRegionalSettingsByAppEnvParams{
		AppID: env.appID, EnvironmentID: env.environmentID,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "concurrent replaces must not merge into multiple regions")

	// The surviving row points at an autoscaling policy that actually exists.
	require.True(t, rows[0].HorizontalAutoscalingPolicyID.Valid, "surviving region must keep a policy")
	var count int
	err = h.DB.RO().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM horizontal_autoscaling_policies WHERE id = ?",
		rows[0].HorizontalAutoscalingPolicyID.String,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "regional row must not dangle: its autoscaling policy must exist")
}
