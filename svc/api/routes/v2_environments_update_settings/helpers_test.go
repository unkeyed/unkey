package handler_test

import (
	"context"
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
		require.NoError(t, db.Query.UpsertRegion(context.Background(), h.DB.RW(), db.UpsertRegionParams{
			ID:       uid.New(uid.RegionPrefix),
			Name:     name,
			Platform: "aws",
		}))
	}
}

// seedUnschedulableRegion inserts an aws region with can_schedule=false. The
// UpsertRegion query has no can_schedule arg, so write it directly.
func seedUnschedulableRegion(t *testing.T, h *testutil.Harness, name string) {
	t.Helper()
	_, err := h.DB.RW().ExecContext(context.Background(),
		"INSERT INTO regions (id, name, platform, can_schedule) VALUES (?, ?, 'aws', false)",
		uid.New(uid.RegionPrefix), name)
	require.NoError(t, err)
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

func regionSetting(name string, minReplicas, maxReplicas int) openapi.RegionSetting {
	return openapi.RegionSetting{
		Name:     name,
		Platform: ptr("aws"),
		Replicas: struct {
			Max int `json:"max"`
			Min int `json:"min"`
		}{Max: maxReplicas, Min: minReplicas},
	}
}
