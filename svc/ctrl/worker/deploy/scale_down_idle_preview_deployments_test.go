package deploy_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestScaleDownIdlePreviewDeployments_ScalesDownIdleDeploymentWithZeroRequests(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	triggerScaleDown(t, h)

	updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsDesiredStateStandby, updated.DesiredState)
}

func TestScaleDownIdlePreviewDeployments_DoesNotScaleDownDeploymentWithRecentRequests(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	h.ClickHouseSeed.InsertSentinelRequests(h.Ctx, ws.ID, project.ID, env.ID, dep.ID, 10, time.Now().Add(-1*time.Hour))

	triggerScaleDown(t, h)

	updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsDesiredStateRunning, updated.DesiredState)
}

func TestScaleDownIdlePreviewDeployments_IgnoresNonPreviewEnvironments(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	triggerScaleDown(t, h)

	updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsDesiredStateRunning, updated.DesiredState)
}

func TestScaleDownIdlePreviewDeployments_IgnoresDeploymentsNotInReadyStatus(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusPending,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	triggerScaleDown(t, h)

	updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsDesiredStateRunning, updated.DesiredState)
}

func TestScaleDownIdlePreviewDeployments_IgnoresRecentlyCreatedDeployments(t *testing.T) {
	h := harness.New(t)

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	recentTime := time.Now().Add(-1 * time.Hour).UnixMilli()
	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     recentTime,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: recentTime},
	})

	triggerScaleDown(t, h)

	updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsDesiredStateRunning, updated.DesiredState)
}

func TestScaleDownIdlePreviewDeployments_IgnoresRecentlyUpdatedDeployments(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	recentUpdate := time.Now().Add(-1 * time.Hour).UnixMilli()
	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: recentUpdate},
	})

	triggerScaleDown(t, h)

	updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeploymentsDesiredStateRunning, updated.DesiredState)
}

func TestScaleDownIdlePreviewDeployments_HandlesMultipleDeploymentsAcrossMultipleEnvironments(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	var idleDeployments []db.Deployment
	var activeDeployments []db.Deployment

	for range 2 {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      ws.ID,
			Name:             "test-project",
			Slug:             uid.New("slug"),
			GitRepositoryURL: "",
			DefaultBranch:    "main",
			DeleteProtection: false,
		})
		env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
			ID:               uid.New(uid.EnvironmentPrefix),
			WorkspaceID:      ws.ID,
			ProjectID:        project.ID,
			Slug:             "preview",
			Description:      "",
			SentinelConfig:   nil,
			DeleteProtection: false,
		})

		idle := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   ws.ID,
			ProjectID:     project.ID,
			EnvironmentID: env.ID,
			Status:        db.DeploymentsStatusReady,
			CreatedAt:     oldTime,
			UpdatedAt:     oldUpdatedAt,
		})
		idleDeployments = append(idleDeployments, idle)

		active := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   ws.ID,
			ProjectID:     project.ID,
			EnvironmentID: env.ID,
			Status:        db.DeploymentsStatusReady,
			CreatedAt:     oldTime,
			UpdatedAt:     oldUpdatedAt,
		})
		h.ClickHouseSeed.InsertSentinelRequests(h.Ctx, ws.ID, project.ID, env.ID, active.ID, 5, time.Now().Add(-30*time.Minute))
		activeDeployments = append(activeDeployments, active)
	}

	triggerScaleDown(t, h)

	for _, dep := range idleDeployments {
		updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
		require.NoError(t, err)
		require.Equal(t, db.DeploymentsDesiredStateStandby, updated.DesiredState, "idle deployment %s should be standby", dep.ID)
	}
	for _, dep := range activeDeployments {
		updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), dep.ID)
		require.NoError(t, err)
		require.Equal(t, db.DeploymentsDesiredStateRunning, updated.DesiredState, "active deployment %s should be running", dep.ID)
	}
}
func TestScaleDownIdlePreviewDeployments_PaginatesAcrossManyPreviewEnvironmentsAtScale(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-8 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	const count = 2000
	depIDs := make([]string, count)
	for i := range count {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      ws.ID,
			Name:             "test-project",
			Slug:             uid.New("slug"),
			GitRepositoryURL: "",
			DefaultBranch:    "main",
			DeleteProtection: false,
		})
		env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
			ID:               uid.New(uid.EnvironmentPrefix),
			WorkspaceID:      ws.ID,
			ProjectID:        project.ID,
			Slug:             "preview",
			Description:      "",
			SentinelConfig:   nil,
			DeleteProtection: false,
		})

		dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   project.WorkspaceID,
			ProjectID:     project.ID,
			EnvironmentID: env.ID,
			Status:        db.DeploymentsStatusReady,
			CreatedAt:     oldTime,
			UpdatedAt:     oldUpdatedAt,
		})
		depIDs[i] = dep.ID
	}

	triggerScaleDown(t, h)

	for _, id := range depIDs {
		updated, err := db.Query.FindDeploymentById(h.Ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, db.DeploymentsDesiredStateStandby, updated.DesiredState, "deployment %s should be standby", id)
	}
}

func triggerScaleDown(t *testing.T, h *harness.Harness) {
	client := hydrav1.NewDeployServiceIngressClient(h.Restate, uid.New("test"))
	_, err := client.ScaleDownIdlePreviewDeployments().Request(h.Ctx, &hydrav1.ScaleDownIdlePreviewDeploymentsRequest{})
	require.NoError(t, err)
}
