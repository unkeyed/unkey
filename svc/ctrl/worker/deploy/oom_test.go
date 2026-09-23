package deploy_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/cluster"
	"github.com/unkeyed/unkey/svc/ctrl/worker/buildslot"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// An OOM can arrive before the pod's instance row. It must still fail the
// rollout, retain actionable advice, and unblock the next deployment without CH.
func TestDeployFirstOOMFailsAndReleasesSlot(t *testing.T) {
	testDeployStartupFailure(t, &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{
		Reason: "OOMKilled", ExitCode: 137, Signal: 9, Cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED,
	}}, hydrav1.NotifyReadinessRequest_STATE_OUT_OF_MEMORY,
		"instance OOMKilled during startup",
		"Your app ran out of memory during startup. Increase the memory limit or reduce startup memory use, then redeploy.")
}

func TestDeployCrashLoopFailsAndReleasesSlot(t *testing.T) {
	testDeployStartupFailure(t, &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{
		Cause: ctrlv1.WaitingCause_WAITING_CAUSE_CRASH_LOOP_BACK_OFF,
	}}, hydrav1.NotifyReadinessRequest_STATE_CRASH_LOOP_BACK_OFF,
		"instance CrashLoopBackOff during startup",
		"Your app repeatedly exited during startup. Check the runtime logs and start command, then redeploy.")
}

func TestDeployContainerCannotRunFailsAndReleasesSlot(t *testing.T) {
	testDeployStartupFailure(t, &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{
		ExitCode: 127, Cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_CONTAINER_CANNOT_RUN,
	}}, hydrav1.NotifyReadinessRequest_STATE_CONTAINER_CANNOT_RUN,
		"instance ContainerCannotRun during startup",
		"Your app could not start. Check the image's entrypoint, executable permissions, and CPU architecture, then redeploy.")
}

func TestDeployReadinessSucceedsAfterContainerConfigError(t *testing.T) {
	testDeployStartupFailure(t, &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{
		Cause: ctrlv1.WaitingCause_WAITING_CAUSE_CONTAINER_CONFIG_ERROR,
	}}, hydrav1.NotifyReadinessRequest_STATE_READY, "", "")
}

func TestDeployInvalidImageNameFailsAndReleasesSlot(t *testing.T) {
	testDeployStartupFailure(t, &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{
		Cause: ctrlv1.WaitingCause_WAITING_CAUSE_INVALID_IMAGE_NAME,
	}}, hydrav1.NotifyReadinessRequest_STATE_INVALID_IMAGE_NAME,
		"instance InvalidImageName during startup",
		"The container image reference is invalid. Check the image name and tag or digest, then redeploy.")
}

func testDeployStartupFailure(t *testing.T, state ctrlv1.IsInstanceEvent_State, notification hydrav1.NotifyReadinessRequest_State, internalError, publicError string) {
	t.Helper()
	ctx := t.Context()
	database, fixture := newDeployFixture(t, ctx)
	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.local",
		GitHub:        githubclient.NewNoop(),
		Build:         deploy.BuildConfig{Backend: deploy.BuildBackendDepot},
		BuildSteps:    batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs: batch.NewNoop[schema.BuildStepLogV1](),
		ImageResolver: deploy.ImageResolverFunc(func(context.Context, string) (string, error) {
			return "example.com/app@sha256:0000000000000000000000000000000000000000000000000000000000000000", nil
		}),
	})
	require.NoError(t, err)
	slots := buildslot.New(buildslot.Config{DB: database, RestateAdmin: oomTestLiveness{}})
	runtime := containers.Restate(t,
		hydrav1.NewDeployWorkflowServer(workflow),
		hydrav1.NewBuildSlotServiceServer(slots),
	)
	admin := restateadmin.New(restateadmin.Config{BaseURL: runtime.AdminURL})
	topologyCache, err := cache.New(cache.Config[string, []db.FindDeploymentTopologyMinReplicasRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 10, Resource: uid.New("oom-test"),
		Clock: clock.New(),
	})
	require.NoError(t, err)
	clusterSvc, err := cluster.New(cluster.Config{
		Database: database, Restate: runtime.IngressClient, RestateAdmin: admin,
		Bearer: "test-token", TopologyCache: topologyCache,
		InstanceEvents: batch.NewNoop[schema.InstanceEventV1](),
	})
	require.NoError(t, err)
	region, err := database.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: "k8s", Name: "kebap-1",
	})
	require.NoError(t, err)
	cellID := uid.New("cell")
	require.NoError(t, database.UpsertCluster(ctx, db.UpsertClusterParams{
		ID: uid.New("cluster"), CellID: sql.NullString{String: cellID, Valid: true},
		RegionID: region.ID, LastHeartbeatAt: uint64(time.Now().UnixMilli()),
	}))
	_, err = database.RW().ExecContext(ctx, "UPDATE `limits` SET builds_concurrent_max = 1 WHERE workspace_id = ?", fixture.workspaceID)
	require.NoError(t, err)

	start := func() db.Deployment {
		t.Helper()
		row := fixture.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			WorkspaceID: fixture.workspaceID, ProjectID: fixture.projectID,
			AppID: fixture.appID, EnvironmentID: fixture.environmentID,
			Status: mysqltype.DeploymentsStatusPending,
		})
		invocation, sendErr := hydrav1.NewDeployWorkflowIngressClient(runtime.IngressClient, row.ID).Submit(ctx, &hydrav1.DeployRequest{
			DeploymentId: row.ID,
			Source:       &hydrav1.DeployRequest_OciImage{OciImage: &hydrav1.OciImage{Image: "example.com/app:v1"}},
		})
		require.NoError(t, sendErr)
		require.NoError(t, database.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
			ID: row.ID, InvocationID: sql.NullString{String: invocation.Id(), Valid: true},
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		}))
		return row
	}
	waitStatus := func(id string, status mysqltype.DeploymentsStatus) {
		t.Helper()
		require.Eventually(t, func() bool {
			row, findErr := database.FindDeploymentById(ctx, id)
			require.NoError(t, findErr)
			return row.Status == status
		}, 15*time.Second, 50*time.Millisecond, "deployment %s should reach %s", id, status)
	}

	first := start()
	waitStatus(first.ID, mysqltype.DeploymentsStatusDeploying)
	next := start()
	client := hydrav1.NewDeployWorkflowIngressClient(runtime.IngressClient, first.ID)
	for _, state := range []hydrav1.NotifyReadinessRequest_State{
		hydrav1.NotifyReadinessRequest_STATE_UNSPECIFIED,
		hydrav1.NotifyReadinessRequest_State(99),
	} {
		_, err = client.NotifyReadiness().Request(ctx, &hydrav1.NotifyReadinessRequest{State: state})
		require.ErrorContains(t, err, "unsupported readiness state")
	}
	event := &ctrlv1.InstanceEvent{
		WorkspaceId: fixture.workspaceID, ProjectId: fixture.projectID,
		AppId: fixture.appID, EnvironmentId: fixture.environmentID,
		DeploymentId: first.ID, PodName: first.K8sName + "-pod", PodUid: uid.New("pod"),
		ContainerName: "deployment", RestartCount: 0, Time: time.Now().UnixMilli(),
		State: state,
	}
	req := connect.NewRequest(&ctrlv1.ReportInstanceEventsRequest{
		Cluster: &ctrlv1.ClusterKey{CellId: cellID, Platform: region.Platform, Region: region.Name},
		Events:  []*ctrlv1.InstanceEvent{event},
	})
	req.Header().Set("Authorization", "Bearer test-token")
	_, err = clusterSvc.ReportInstanceEvents(ctx, req)
	require.NoError(t, err)
	if notification == hydrav1.NotifyReadinessRequest_STATE_READY {
		_, err = client.NotifyReadiness().Request(ctx, &hydrav1.NotifyReadinessRequest{State: hydrav1.NotifyReadinessRequest_STATE_CONTAINER_CONFIG_ERROR})
		require.NoError(t, err)
		_, err = client.NotifyReadiness().Request(ctx, &hydrav1.NotifyReadinessRequest{State: notification})
		require.NoError(t, err)
		var failure sql.NullString
		require.Eventually(t, func() bool {
			var endedAt sql.NullInt64
			queryErr := database.RW().QueryRowContext(ctx,
				"SELECT error, ended_at FROM deployment_steps WHERE deployment_id = ? AND step = 'deploying'", first.ID,
			).Scan(&failure, &endedAt)
			require.NoError(t, queryErr)
			return endedAt.Valid
		}, 15*time.Second, 50*time.Millisecond)
		require.False(t, failure.Valid, "READY must finish the deploying step successfully after a transient configuration error")
		return
	}
	waitStatus(first.ID, mysqltype.DeploymentsStatusFailed)

	_, err = client.NotifyReadiness().Request(ctx, &hydrav1.NotifyReadinessRequest{State: notification})
	require.NoError(t, err, "duplicate failure notifications must be harmless")
	_, err = client.NotifyReadiness().Request(ctx, &hydrav1.NotifyReadinessRequest{State: hydrav1.NotifyReadinessRequest_STATE_READY})
	require.NoError(t, err, "late readiness must not overwrite the failure")
	_, err = client.NotifyInstancesReady().Request(ctx, &hydrav1.NotifyInstancesReadyRequest{DeploymentId: first.ID})
	require.NoError(t, err, "legacy readiness must not overwrite the failure")
	_, err = client.Handle().Attach(ctx)
	require.ErrorContains(t, err, internalError)

	var failure string
	var endedAt sql.NullInt64
	require.NoError(t, database.RW().QueryRowContext(ctx,
		"SELECT error, ended_at FROM deployment_steps WHERE deployment_id = ? AND step = 'deploying'", first.ID,
	).Scan(&failure, &endedAt))
	require.True(t, endedAt.Valid)
	require.Equal(t, publicError, failure)
	topology, err := database.FindDeploymentTopologyByDeploymentAndRegion(ctx, db.FindDeploymentTopologyByDeploymentAndRegionParams{
		DeploymentID: first.ID, RegionID: region.ID,
	})
	require.NoError(t, err)
	require.Equal(t, db.DeploymentTopologyDesiredStatusStopped, topology.DesiredStatus)
	waitStatus(next.ID, mysqltype.DeploymentsStatusDeploying)
}

type oomTestLiveness struct{}

func (oomTestLiveness) FindLiveInvocations(_ context.Context, ids []string) (map[string]bool, error) {
	live := make(map[string]bool, len(ids))
	for _, id := range ids {
		live[id] = true
	}
	return live, nil
}
