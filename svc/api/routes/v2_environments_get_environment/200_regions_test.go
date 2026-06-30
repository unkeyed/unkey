package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_get_environment"
)

func TestGetEnvironmentRegions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_environment")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})

	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	platform := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	require.NoError(t, db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
		ID:       uid.New(uid.RegionPrefix),
		Name:     "us-east-1",
		Platform: platform,
	}))
	region, err := db.Query.FindRegionByPlatformAndName(ctx, h.DB.RO(), db.FindRegionByPlatformAndNameParams{
		Name:     "us-east-1",
		Platform: platform,
	})
	require.NoError(t, err)
	regionID := region.ID

	now := time.Now().UnixMilli()

	t.Run("without autoscaling policy uses replica count as min and max", func(t *testing.T) {
		environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        "production",
			Description: "Production environment",
		})

		require.NoError(t, db.Query.UpsertAppRegionalSettings(ctx, h.DB.RW(), db.UpsertAppRegionalSettingsParams{
			WorkspaceID:   workspace.ID,
			AppID:         app.ID,
			EnvironmentID: environment.ID,
			RegionID:      regionID,
			Replicas:      3,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Valid: false},
		}))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:     project.ID,
			App:         app.ID,
			Environment: environment.ID,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Data.Regions)
		regions := *res.Body.Data.Regions
		require.Len(t, regions, 1)
		require.Equal(t, "us-east-1", regions[0].Name)
		require.Equal(t, 3, regions[0].Replicas.Min)
		require.Equal(t, 3, regions[0].Replicas.Max)
	})

	t.Run("with autoscaling policy uses policy bounds", func(t *testing.T) {
		environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        "preview",
			Description: "Preview environment",
		})

		require.NoError(t, db.Query.UpsertAppRegionalSettings(ctx, h.DB.RW(), db.UpsertAppRegionalSettingsParams{
			WorkspaceID:   workspace.ID,
			AppID:         app.ID,
			EnvironmentID: environment.ID,
			RegionID:      regionID,
			Replicas:      2,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Valid: false},
		}))

		// No typed query for autoscaling policies on this branch; insert raw and
		// point the regional setting at it.
		policyID := uid.New("hap")
		_, err := h.DB.RW().ExecContext(ctx,
			"INSERT INTO horizontal_autoscaling_policies (id, workspace_id, replicas_min, replicas_max, cpu_threshold, memory_threshold, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			policyID, workspace.ID, 1, 5, 80, 80, now)
		require.NoError(t, err)
		_, err = h.DB.RW().ExecContext(ctx,
			"UPDATE app_regional_settings SET horizontal_autoscaling_policy_id = ? WHERE app_id = ? AND environment_id = ? AND region_id = ?",
			policyID, app.ID, environment.ID, regionID)
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:     project.ID,
			App:         app.ID,
			Environment: environment.ID,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Data.Regions)
		regions := *res.Body.Data.Regions
		require.Len(t, regions, 1)
		require.Equal(t, "us-east-1", regions[0].Name)
		require.Equal(t, 1, regions[0].Replicas.Min)
		require.Equal(t, 5, regions[0].Replicas.Max)
	})
}
