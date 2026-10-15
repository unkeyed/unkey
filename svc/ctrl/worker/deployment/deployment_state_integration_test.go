package deployment_test

import (
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
)

func seedDesiredStateDeployment(t *testing.T, h *harness.Harness) db.Deployment {
	t.Helper()
	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})
	return h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
}

func TestChangeDesiredState_NoOpsWhenDeploymentDeleted(t *testing.T) {
	h := harness.New(t)
	dep := seedDesiredStateDeployment(t, h)
	client := hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID)
	_, err := client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 500,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)

	require.NoError(t, h.DB.DeleteDeploymentTopologiesByEnvironmentId(h.Ctx, dep.EnvironmentID))
	require.NoError(t, h.DB.DeleteDeploymentsByEnvironmentId(h.Ctx, dep.EnvironmentID))

	time.Sleep(time.Second)

	_, err = h.DB.FindDeploymentById(h.Ctx, dep.ID)
	require.Error(t, err)
	require.True(t, db.IsNotFound(err))

	h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID: dep.ID, WorkspaceID: dep.WorkspaceID, ProjectID: dep.ProjectID,
		AppID: dep.AppID, EnvironmentID: dep.EnvironmentID, Status: mysqltype.DeploymentsStatusReady,
	})
	_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE deployments SET desired_state = 'stopped' WHERE id = ?", dep.ID)
	require.NoError(t, err)
	_, err = client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 0,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING,
		Overwrite:   false,
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		deployment, readErr := h.DB.FindDeploymentById(h.Ctx, dep.ID)
		return readErr == nil && deployment.DesiredState == mysqltype.DeploymentsDesiredStateRunning
	}, 10*time.Second, 25*time.Millisecond, "a missing deployment must clear its pending transition")
}

func TestQueuedDesiredStateFencesMarkedAncestorsAndClearsTransition(t *testing.T) {
	h := harness.New(t)
	applyRuntime := containers.Restate(t, restate.NewObject("test.ApplyDesiredState").Handler("Apply", restate.NewObjectHandler(func(ctx restate.ObjectContext, _ restate.Void) (restate.Void, error) {
		return restate.Void{}, deployment.ApplyDesiredState(ctx, h.DB, restate.Key(ctx), mysqltype.DeploymentsDesiredStateStopped, db.DeploymentTopologyDesiredStatusStopped)
	})))
	for _, ancestor := range []string{"live", "environments", "apps", "projects"} {
		t.Run(ancestor, func(t *testing.T) {
			dep := seedDesiredStateDeployment(t, h)
			region := h.Seed.CreateRegion(h.Ctx, seed.CreateRegionRequest{Name: uid.DNS1035(), Platform: "test"})
			_, err := h.DB.RW().ExecContext(h.Ctx, "UPDATE deployments SET desired_state = 'running' WHERE id = ?", dep.ID)
			require.NoError(t, err)
			require.NoError(t, h.DB.InsertDeploymentTopology(h.Ctx, db.InsertDeploymentTopologyParams{
				WorkspaceID: dep.WorkspaceID, DeploymentID: dep.ID, RegionID: region.ID,
				DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: time.Now().UnixMilli(),
				AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
			}))
			client := hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID)
			_, err = client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: 500, State: hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED, Overwrite: true,
			})
			require.NoError(t, err)
			ancestorID := map[string]string{"environments": dep.EnvironmentID, "apps": dep.AppID, "projects": dep.ProjectID}[ancestor]
			if ancestor != "live" {
				_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE "+ancestor+" SET deleted_at = ? WHERE id = ?", time.Now().UnixMilli(), ancestorID)
				require.NoError(t, err)
			}
			time.Sleep(time.Second)
			var desiredState, topologyStatus string
			require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx, "SELECT desired_state FROM deployments WHERE id = ?", dep.ID).Scan(&desiredState))
			require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx, "SELECT desired_status FROM deployment_topology WHERE deployment_id = ? AND region_id = ?", dep.ID, region.ID).Scan(&topologyStatus))
			want := "running"
			if ancestor == "live" {
				want = "stopped"
			}
			require.Equal(t, want, desiredState)
			require.Equal(t, want, topologyStatus)
			if ancestor == "live" {
				_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE deployments SET desired_state = 'running' WHERE id = ?", dep.ID)
				require.NoError(t, err)
				_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE deployment_topology SET desired_status = 'running' WHERE deployment_id = ?", dep.ID)
				require.NoError(t, err)
			}
			_, err = ingress.Object[restate.Void, restate.Void](applyRuntime.IngressClient, "test.ApplyDesiredState", dep.ID, "Apply").Request(h.Ctx, restate.Void{})
			if ancestor == "live" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx, "SELECT desired_state FROM deployments WHERE id = ?", dep.ID).Scan(&desiredState))
			require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx, "SELECT desired_status FROM deployment_topology WHERE deployment_id = ? AND region_id = ?", dep.ID, region.ID).Scan(&topologyStatus))
			require.Equal(t, want, desiredState)
			require.Equal(t, want, topologyStatus)

			if ancestor != "live" {
				_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE "+ancestor+" SET deleted_at = NULL WHERE id = ?", ancestorID)
				require.NoError(t, err)
			}
			_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE deployments SET desired_state = 'stopped' WHERE id = ?", dep.ID)
			require.NoError(t, err)
			_, err = h.DB.RW().ExecContext(h.Ctx, "UPDATE deployment_topology SET desired_status = 'stopped' WHERE deployment_id = ?", dep.ID)
			require.NoError(t, err)
			_, err = client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
				State: hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING, Overwrite: false,
			})
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				queryErr := h.DB.RW().QueryRowContext(h.Ctx, "SELECT desired_state FROM deployments WHERE id = ?", dep.ID).Scan(&desiredState)
				if queryErr != nil {
					return false
				}
				queryErr = h.DB.RW().QueryRowContext(h.Ctx, "SELECT desired_status FROM deployment_topology WHERE deployment_id = ? AND region_id = ?", dep.ID, region.ID).Scan(&topologyStatus)
				return queryErr == nil && desiredState == "running" && topologyStatus == "running"
			}, 10*time.Second, 25*time.Millisecond, "completed transitions must not block a non-overwriting schedule")
		})
	}
}
