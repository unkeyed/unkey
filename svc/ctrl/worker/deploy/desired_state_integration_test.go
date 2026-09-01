package deploy_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// TestChangeDesiredStateNoOpsWhenDeploymentDeleted: a delayed
// ChangeDesiredState whose deployment row was cascade-deleted after
// scheduling must succeed as a no-op instead of retrying forever.
func TestChangeDesiredStateNoOpsWhenDeploymentDeleted(t *testing.T) {
	ctx := context.Background()
	h := newDesiredStateHarness(t, ctx)

	dep := h.seedDeployment(t, ctx, mysqltype.DeploymentsDesiredStateRunning)

	client := hydrav1.NewDeployServiceIngressClient(h.ingress, dep.ID)
	_, err := client.ScheduleDesiredStateChange().Request(ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 500,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)

	require.NoError(t, h.database.DeleteDeploymentTopologiesByEnvironmentId(ctx, h.environmentID))
	require.NoError(t, h.database.DeleteDeploymentsByEnvironmentId(ctx, h.environmentID))

	time.Sleep(time.Second)

	_, err = h.database.FindDeploymentById(ctx, dep.ID)
	require.Error(t, err)
	require.True(t, db.IsNotFound(err))

	_, err = client.ScheduleDesiredStateChange().Request(ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 0,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)
}

// TestStopDeploymentAppliesStateInline pins the stop contract: when the
// request returns, the deployment's desired state, its topology rows, and the
// change feed all record the stop. The apply is inline, so no waiting.
func TestStopDeploymentAppliesStateInline(t *testing.T) {
	ctx := context.Background()
	h := newDesiredStateHarness(t, ctx)

	dep := h.seedDeployment(t, ctx, mysqltype.DeploymentsDesiredStateRunning)

	_, err := hydrav1.NewDeployServiceIngressClient(h.ingress, dep.ID).
		StopDeployment().
		Request(ctx, &hydrav1.StopDeploymentRequest{
			DeploymentId:  dep.ID,
			Actor:         kebapActor(),
			CorrelationId: "",
		})
	require.NoError(t, err)

	after, err := h.database.FindDeploymentById(ctx, dep.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateStopped, after.DesiredState)

	require.Equal(t, string(db.DeploymentTopologyDesiredStatusStopped), h.topologyDesiredStatus(t, ctx, dep.ID))
	require.Positive(t, h.countDeploymentChanges(t, ctx, dep.ID),
		"the stop must emit a deployment_changes row so regions pick it up")
}

// TestWakeDeploymentSupersedesPendingStop is the scenario the inline apply
// exists for: a stop is already scheduled when a user wakes the deployment.
// The wake supersedes the pending transition, so the delayed
// ChangeDesiredState fires, finds nothing, and the deployment stays running.
func TestWakeDeploymentSupersedesPendingStop(t *testing.T) {
	ctx := context.Background()
	h := newDesiredStateHarness(t, ctx)

	dep := h.seedDeployment(t, ctx, mysqltype.DeploymentsDesiredStateStopped)

	const standbyDelay = 2 * time.Second
	_, err := hydrav1.NewDeployServiceIngressClient(h.ingress, dep.ID).
		ScheduleDesiredStateChange().
		Request(ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
			DelayMillis: standbyDelay.Milliseconds(),
			State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
			Overwrite:   true,
		})
	require.NoError(t, err)

	_, err = hydrav1.NewDeployServiceIngressClient(h.ingress, dep.ID).
		WakeDeployment().
		Request(ctx, &hydrav1.WakeDeploymentRequest{
			DeploymentId:  dep.ID,
			Actor:         kebapActor(),
			CorrelationId: "",
		})
	require.NoError(t, err)

	// Wait out the scheduled delay plus margin so the stale delayed call has
	// definitely fired before asserting it changed nothing.
	time.Sleep(standbyDelay + 1500*time.Millisecond)

	after, err := h.database.FindDeploymentById(ctx, dep.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateRunning, after.DesiredState,
		"the pending stop must not fire against the woken deployment")
	require.Equal(t, mysqltype.DeploymentsStatusReady, after.Status)
	require.Equal(t, string(db.DeploymentTopologyDesiredStatusRunning), h.topologyDesiredStatus(t, ctx, dep.ID))
}

// desiredStateHarness is one MySQL database and one Restate server hosting
// the real deploy workflow.
type desiredStateHarness struct {
	database db.Database
	seeder   *seed.Seeder
	ingress  *restateingress.Client

	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	regionID      string
}

func newDesiredStateHarness(t *testing.T, ctx context.Context) *desiredStateHarness {
	t.Helper()

	database, fixture := newDeployFixture(t, ctx)

	// Stop and wake refuse production targets, so these tests run on a
	// preview environment rather than the fixture's production one.
	previewEnv := fixture.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: fixture.workspaceID,
		ProjectID:   fixture.projectID,
		AppID:       fixture.appID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})

	region := fixture.seeder.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     "kebap-1",
		Platform: "aws",
	})

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.com",
		Vault:         nil,
		GitHub:        nil,
		Build: deploy.BuildConfig{
			Backend:    deploy.BuildBackendDepot,
			Depot:      deploy.DepotConfig{APIUrl: "", ProjectRegion: "", ProjectPrefix: "builds-test"},
			Kubernetes: deploy.KubernetesBuildConfig{Namespace: "", Image: ""},
		},
		K8s:                             nil,
		RegistryConfig:                  deploy.RegistryConfig{Repository: "", Username: "", Password: "", Insecure: false},
		BuildPlatform:                   deploy.BuildPlatform{Platform: "", Architecture: ""},
		Clickhouse:                      nil,
		BuildSteps:                      batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs:                   batch.NewNoop[schema.BuildStepLogV1](),
		AllowUnauthenticatedDeployments: false,
	})
	require.NoError(t, err)

	restateCfg := containers.Restate(t, hydrav1.NewDeployServiceServer(workflow))

	return &desiredStateHarness{
		database:      database,
		seeder:        fixture.seeder,
		ingress:       restateCfg.IngressClient,
		workspaceID:   fixture.workspaceID,
		projectID:     fixture.projectID,
		appID:         fixture.appID,
		environmentID: previewEnv.ID,
		regionID:      region.ID,
	}
}

// seedDeployment creates a ready preview deployment with one topology row and
// one running instance, so a wake's health poll succeeds immediately.
func (h *desiredStateHarness) seedDeployment(t *testing.T, ctx context.Context, desired mysqltype.DeploymentsDesiredState) db.Deployment {
	t.Helper()

	dep := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   h.workspaceID,
		ProjectID:     h.projectID,
		AppID:         h.appID,
		EnvironmentID: h.environmentID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	topologyStatus := db.DeploymentTopologyDesiredStatusRunning
	if desired == mysqltype.DeploymentsDesiredStateStopped {
		topologyStatus = db.DeploymentTopologyDesiredStatusStopped
		require.NoError(t, h.database.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
			ID:           dep.ID,
			DesiredState: desired,
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))
	}

	require.NoError(t, h.database.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
		WorkspaceID:            h.workspaceID,
		DeploymentID:           dep.ID,
		RegionID:               h.regionID,
		AutoscalingReplicasMin: 1,
		AutoscalingReplicasMax: 1,
		DesiredStatus:          topologyStatus,
		CreatedAt:              time.Now().UnixMilli(),
	}))

	h.seeder.CreateInstance(ctx, seed.CreateInstanceRequest{
		DeploymentID: dep.ID,
		WorkspaceID:  h.workspaceID,
		ProjectID:    h.projectID,
		AppID:        h.appID,
		RegionID:     h.regionID,
		Address:      "10.0.0.1:8080",
	})

	return dep
}

func (h *desiredStateHarness) topologyDesiredStatus(t *testing.T, ctx context.Context, deploymentID string) string {
	t.Helper()
	var status string
	require.NoError(t, h.database.RO().QueryRowContext(ctx,
		"SELECT desired_status FROM deployment_topology WHERE deployment_id = ? AND region_id = ?",
		deploymentID, h.regionID,
	).Scan(&status))
	return status
}

func (h *desiredStateHarness) countDeploymentChanges(t *testing.T, ctx context.Context, deploymentID string) int {
	t.Helper()
	var count int
	require.NoError(t, h.database.RO().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployment_changes WHERE resource_id = ?", deploymentID,
	).Scan(&count))
	return count
}

func kebapActor() *ctrlv1.ActorInfo {
	return &ctrlv1.ActorInfo{
		Id:        "user_KEBAP",
		Name:      "KEBAP",
		Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
		RemoteIp:  "",
		UserAgent: "",
		Meta:      nil,
	}
}
