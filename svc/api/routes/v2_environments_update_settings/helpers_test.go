package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// seededEnv holds the ids of a freshly seeded project/app/environment.
// CreateEnvironment also seeds default build and runtime settings rows, so the
// handler's UPDATE statements always have a row to target.
type seededEnv struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func seedEnvironment(t *testing.T, h *testutil.Harness) seededEnv {
	t.Helper()

	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
	})

	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
		DefaultBranch: "main",
	})

	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	return seededEnv{
		workspaceID:   workspace.ID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

// seedRegions inserts schedulable aws regions by name for tests.
func seedRegions(t *testing.T, h *testutil.Harness, names ...string) {
	t.Helper()
	for _, name := range names {
		regionID := uid.New(uid.RegionPrefix)
		require.NoError(t, db.Query.UpsertRegion(t.Context(), h.DB.RW(), db.UpsertRegionParams{
			ID:       regionID,
			Name:     name,
			Platform: "aws",
		}))
		region, err := db.Query.FindRegionByPlatformAndName(t.Context(), h.DB.RO(), db.FindRegionByPlatformAndNameParams{
			Platform: "aws",
			Name:     name,
		})
		require.NoError(t, err)
		require.NoError(t, db.Query.UpsertCluster(t.Context(), h.DB.RW(), db.UpsertClusterParams{
			ID:              uid.New(uid.ClusterPrefix),
			RegionID:        region.ID,
			Platform:        "aws",
			Region:          name,
			State:           db.ClustersStateActive,
			LastHeartbeatAt: 1,
		}))
	}
}

// seedUnschedulableRegion inserts an aws region with can_schedule=false. The
// UpsertRegion query has no can_schedule arg, so write it directly.
func seedUnschedulableRegion(t *testing.T, h *testutil.Harness, name string) {
	t.Helper()
	regionID := uid.New(uid.RegionPrefix)
	_, err := h.DB.RW().ExecContext(t.Context(),
		"INSERT INTO regions (id, name, platform, can_schedule) VALUES (?, ?, 'aws', false) ON DUPLICATE KEY UPDATE can_schedule = false",
		regionID, name)
	require.NoError(t, err)
	region, err := db.Query.FindRegionByPlatformAndName(t.Context(), h.DB.RO(), db.FindRegionByPlatformAndNameParams{
		Platform: "aws",
		Name:     name,
	})
	require.NoError(t, err)
	require.NoError(t, db.Query.UpsertCluster(t.Context(), h.DB.RW(), db.UpsertClusterParams{
		ID:              uid.New(uid.ClusterPrefix),
		RegionID:        region.ID,
		Platform:        "aws",
		Region:          name,
		State:           db.ClustersStateDisabled,
		LastHeartbeatAt: 1,
	}))
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

func ptr[T any](v T) *T {
	return &v
}

func regionSetting(name string, minReplicas, maxReplicas int) openapi.EnvironmentRegion {
	return openapi.EnvironmentRegion{
		Name:     name,
		Replicas: openapi.Replicas{Max: maxReplicas, Min: minReplicas},
	}
}
