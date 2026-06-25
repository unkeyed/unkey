package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_update_settings"
)

func TestUpdateSettingsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	ctx := context.Background()
	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	// Seed a schedulable region and a non-schedulable one.
	schedulableID := uid.New(uid.RegionPrefix)
	require.NoError(t, db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
		ID: schedulableID, Name: "us-east-1", Platform: "aws",
	}))
	blockedID := uid.New(uid.RegionPrefix)
	require.NoError(t, db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
		ID: blockedID, Name: "eu-west-1", Platform: "aws",
	}))
	_, err := h.DB.RW().ExecContext(ctx, "UPDATE regions SET can_schedule = ? WHERE id = ?", false, blockedID)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		regions []openapi.RegionSetting
	}{
		{name: "min greater than max", regions: []openapi.RegionSetting{regionSetting("us-east-1", 3, 1)}},
		{name: "max above limit", regions: []openapi.RegionSetting{regionSetting("us-east-1", 1, 5)}},
		{name: "min below one", regions: []openapi.RegionSetting{regionSetting("us-east-1", 0, 2)}},
		{name: "unknown region", regions: []openapi.RegionSetting{regionSetting("ap-south-1", 1, 2)}},
		{name: "duplicate region", regions: []openapi.RegionSetting{regionSetting("us-east-1", 1, 2), regionSetting("us-east-1", 1, 3)}},
		{name: "non-schedulable region", regions: []openapi.RegionSetting{regionSetting("eu-west-1", 1, 2)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			regions := tc.regions
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
				Project:     env.projectID,
				App:         env.appID,
				Environment: env.environmentID,
				Regions:     &regions,
			})
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 for %q, got: %s", tc.name, res.RawBody)
		})
	}
}
