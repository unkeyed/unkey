//go:build integration

package integration

import (
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	appservice "github.com/unkeyed/unkey/svc/ctrl/services/app"
)

func TestCreateAppInitializesDefaultSettingsAtomically(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const bearer = "test-token"

	auditSvc, err := auditlogs.New(auditlogs.Config{DB: h.DB})
	require.NoError(t, err)

	svc := appservice.New(appservice.Config{
		Database:  h.DB,
		Auditlogs: auditSvc,
		Bearer:    bearer,
	})

	workspaceID := h.Resources().UserWorkspace.ID
	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Payments",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("slug"), "_", "-")),
	})

	schedulableRegion := h.Seed.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     "test-schedulable",
		Platform: "test",
	})
	unschedulableRegion := h.Seed.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     "test-unschedulable",
		Platform: "test",
	})
	_, err = h.DB.RW().ExecContext(
		ctx,
		"UPDATE regions SET can_schedule = false WHERE id = ?",
		unschedulableRegion.ID,
	)
	require.NoError(t, err)

	req := connect.NewRequest(&ctrlv1.CreateAppRequest{
		WorkspaceId: workspaceID,
		ProjectId:   project.ID,
		Name:        "API",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("app"), "_", "-")),
		Actor: &ctrlv1.ActorInfo{
			Id:   "user_test",
			Type: ctrlv1.ActorType_ACTOR_TYPE_USER,
		},
	})
	req.Header().Set("Authorization", "Bearer "+bearer)

	res, err := svc.CreateApp(ctx, req)
	require.NoError(t, err)
	appID := res.Msg.GetId()

	for _, slug := range []string{"production", "preview"} {
		environment, findErr := h.DB.FindEnvironmentByAppIdAndSlug(
			ctx,
			db.FindEnvironmentByAppIdAndSlugParams{AppID: appID, Slug: slug},
		)
		require.NoError(t, findErr)

		runtimeSettings, findErr := h.DB.FindAppRuntimeSettingsByAppAndEnv(
			ctx,
			db.FindAppRuntimeSettingsByAppAndEnvParams{
				AppID:         appID,
				EnvironmentID: environment.Environment.ID,
			},
		)
		require.NoError(t, findErr)
		require.Equal(t, int32(8080), runtimeSettings.AppRuntimeSetting.Port)
		require.Equal(t, int32(250), runtimeSettings.AppRuntimeSetting.CpuMillicores)
		require.Equal(t, int32(256), runtimeSettings.AppRuntimeSetting.MemoryMib)

		regionalSettings, findErr := h.DB.FindAppRegionalSettingsByAppAndEnv(
			ctx,
			db.FindAppRegionalSettingsByAppAndEnvParams{
				AppID:         appID,
				EnvironmentID: environment.Environment.ID,
			},
		)
		require.NoError(t, findErr)
		regionIDs := make([]string, 0, len(regionalSettings))
		for _, regionalSetting := range regionalSettings {
			require.True(t, regionalSetting.RegionCanSchedule)
			require.Equal(t, int32(1), regionalSetting.Replicas)
			regionIDs = append(regionIDs, regionalSetting.RegionID)
		}
		require.Contains(t, regionIDs, schedulableRegion.ID)
		require.NotContains(t, regionIDs, unschedulableRegion.ID)
	}

	_, err = h.DB.RW().ExecContext(ctx, "UPDATE regions SET can_schedule = false")
	require.NoError(t, err)

	invalidAppSlug := strings.ToLower(strings.ReplaceAll(uid.New("app"), "_", "-"))
	invalidReq := connect.NewRequest(&ctrlv1.CreateAppRequest{
		WorkspaceId: workspaceID,
		ProjectId:   project.ID,
		Name:        "Invalid API",
		Slug:        invalidAppSlug,
		Actor: &ctrlv1.ActorInfo{
			Id:   "user_test",
			Type: ctrlv1.ActorType_ACTOR_TYPE_USER,
		},
	})
	invalidReq.Header().Set("Authorization", "Bearer "+bearer)

	_, err = svc.CreateApp(ctx, invalidReq)
	require.Error(t, err)

	var invalidAppCount int
	err = h.DB.RO().QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM apps WHERE project_id = ? AND slug = ?",
		project.ID,
		invalidAppSlug,
	).Scan(&invalidAppCount)
	require.NoError(t, err)
	require.Zero(t, invalidAppCount)
}
