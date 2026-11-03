package deployment_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
)

// The first attempt records the pre-retirement journal prefix, then fails.
// Its retry must finish through ApplyDesiredState without losing regional delivery.
func TestApplyDesiredStateResumesLegacyRegionalJournal(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	row := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix),
		AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
		Status: mysqltype.DeploymentsStatusReady,
	})
	for range 2 {
		region := seeder.CreateRegion(ctx, seed.CreateRegionRequest{Name: uid.DNS1035(12), Platform: "test"})
		require.NoError(t, database.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
			WorkspaceID: row.WorkspaceID, DeploymentID: row.ID, RegionID: region.ID,
			AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1, CreatedAt: 1,
		}))
	}
	var interrupted atomic.Bool
	const service = "unkey.test.IntentReplay"
	worker := restate.NewObject(service).Handler("stop", restate.NewObjectHandler(
		func(ctx restate.ObjectContext, _ string) (string, error) {
			if !interrupted.Swap(true) {
				if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
					return database.UpdateDeploymentDesiredState(runCtx, db.UpdateDeploymentDesiredStateParams{
						ID: row.ID, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
					})
				}, restate.WithName("updating desired state")); err != nil {
					return "", err
				}
				regions, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Region, error) {
					return database.FindDeploymentRegions(runCtx, row.ID)
				}, restate.WithName("find deployment regions"))
				if err != nil {
					return "", err
				}
				if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
					return database.UpdateDeploymentDesiredState(runCtx, db.UpdateDeploymentDesiredStateParams{
						ID: row.ID, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
					})
				}, restate.WithName(fmt.Sprintf("updating topology desired status in %s", regions[0].ID))); err != nil {
					return "", err
				}
				return "", errors.New("worker interrupted after first regional write")
			}
			if err := deployment.ApplyDesiredState(ctx, database, row.ID, mysqltype.DeploymentsDesiredStateStopped); err != nil {
				return "", err
			}
			return "stopped", nil
		}))
	cfg := containers.Restate(t, worker)
	response, err := ingress.Object[string, string](cfg.IngressClient, service, row.ID, "stop").Request(ctx, "")
	require.NoError(t, err)
	require.Equal(t, "stopped", response)
	current, err := database.FindDeploymentById(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateStopped, current.DesiredState)
}

// TestChangeDesiredState_NoOpsWhenDeploymentDeleted verifies that a delayed
// ChangeDesiredState call succeeds when the deployment row was removed after
// scheduling, for example by an environment delete cascade.
func TestChangeDesiredState_NoOpsWhenDeploymentDeleted(t *testing.T) {
	h := harness.New(t)

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
	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	client := hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID)
	_, err := client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 500,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)

	require.NoError(t, h.DB.DeleteDeploymentTopologiesByEnvironmentId(h.Ctx, env.ID))
	require.NoError(t, h.DB.DeleteDeploymentsByEnvironmentId(h.Ctx, env.ID))

	time.Sleep(time.Second)

	_, err = h.DB.FindDeploymentById(h.Ctx, dep.ID)
	require.Error(t, err)
	require.True(t, db.IsNotFound(err))

	_, err = client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 0,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)
}
