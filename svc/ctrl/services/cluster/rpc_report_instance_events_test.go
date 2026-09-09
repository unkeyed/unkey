package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type podFailure struct {
	PodUID     string `json:"podUid"`
	PodName    string `json:"podName"`
	RegionID   string `json:"regionId"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	ObservedAt int64  `json:"observedAt"`
}

func TestReportInstanceEventsRetainsPodFailureAfterInstanceReconciliation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	workspace := seeder.CreateWorkspace(ctx)
	otherWorkspace := seeder.CreateWorkspace(ctx)
	projectID, err := database.FindDefaultProjectByWorkspaceID(ctx, workspace.ID)
	require.NoError(t, err)
	appID := uid.New(uid.AppPrefix)
	seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID: appID, WorkspaceID: workspace.ID, ProjectID: projectID,
		Name: "pod failure test", Slug: uid.DNS1035(), DefaultBranch: "main",
	})
	environmentID := uid.New(uid.EnvironmentPrefix)
	seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID: environmentID, WorkspaceID: workspace.ID, ProjectID: projectID, AppID: appID,
		Slug: "production", Kind: dbtype.EnvironmentKindProduction,
	})
	deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID: workspace.ID, ProjectID: projectID, AppID: appID,
		EnvironmentID: environmentID, Status: dbtype.DeploymentsStatusReady,
	})

	const bearer = "cluster-test-bearer"
	clusterKey := &ctrlv1.ClusterKey{CellId: "cell-test", Platform: "kubernetes", Region: "test-region"}
	service := newInstanceEventTestService(t, database, bearer)
	heartbeat := authenticatedRequest(bearer, &ctrlv1.HeartbeatRequest{Cluster: clusterKey})
	_, err = service.Heartbeat(ctx, heartbeat)
	require.NoError(t, err)
	region, err := database.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: clusterKey.Platform, Name: clusterKey.Region,
	})
	require.NoError(t, err)

	oldPodName := uid.DNS1035()
	t.Cleanup(func() {
		require.NoError(t, database.DeleteDeploymentInstances(context.Background(), db.DeleteDeploymentInstancesParams{
			DeploymentID: deployment.ID, RegionID: region.ID,
		}))
	})
	reportDeploymentUpdate(t, ctx, service, bearer, clusterKey, deployment.K8sName, oldPodName,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING)
	require.Equal(t, 1, instanceCount(t, ctx, database, deployment.ID))

	reportEmptyDeploymentSnapshot(t, ctx, service, bearer, clusterKey, deployment.K8sName)
	require.Equal(t, 0, instanceCount(t, ctx, database, deployment.ID))

	const (
		failureObservedAt = int64(1_789_000_123_456)
		failureReason     = "Evicted"
		failureMessage    = "The node was low on ephemeral-storage; container logs remain available"
	)
	eviction := instanceEvent(deployment, workspace.ID, oldPodName, failureObservedAt, failureReason, failureMessage)
	eviction.PodUid = "pod-uid-evicted"
	eviction.Attributes = map[string]string{"pod_phase": "Pending"}
	reportEvents(t, ctx, service, bearer, clusterKey, eviction)
	require.Equal(t, 0, instanceCount(t, ctx, database, deployment.ID))

	wantFailure := podFailure{
		PodUID: eviction.PodUid, PodName: oldPodName, RegionID: region.ID,
		Reason: failureReason, Message: failureMessage, ObservedAt: failureObservedAt,
	}
	require.Equal(t, wantFailure, readPodFailure(t, ctx, database, deployment.ID))

	stale := instanceEvent(deployment, workspace.ID, "stale-pod", failureObservedAt-1, "Failed", "stale failure")
	reportEvents(t, ctx, service, bearer, clusterKey, stale)
	repeated := instanceEvent(deployment, workspace.ID, "repeated-pod", failureObservedAt, "Failed", "same timestamp")
	reportEvents(t, ctx, service, bearer, clusterKey, repeated)
	crossWorkspace := instanceEvent(deployment, otherWorkspace.ID, "other-tenant-pod", failureObservedAt+1_000, "Failed", "other tenant")
	reportEvents(t, ctx, service, bearer, clusterKey, crossWorkspace)
	require.Equal(t, wantFailure, readPodFailure(t, ctx, database, deployment.ID))

	replacementPodName := uid.DNS1035()
	reportDeploymentUpdate(t, ctx, service, bearer, clusterKey, deployment.K8sName, replacementPodName,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING)
	require.Equal(t, []string{replacementPodName}, instanceNames(t, ctx, database, deployment.ID))

	imagePull := instanceEvent(deployment, workspace.ID, replacementPodName, failureObservedAt+2_000, "ImagePullBackOff", "registry unavailable")
	imagePull.Attributes = map[string]string{"pod_phase": "Pending"}
	reportEvents(t, ctx, service, bearer, clusterKey, imagePull)
	require.Equal(t, "ImagePullBackOff", waitingReason(t, ctx, database, replacementPodName, region.ID))
	require.Equal(t, wantFailure, readPodFailure(t, ctx, database, deployment.ID))

	running := instanceEvent(deployment, workspace.ID, replacementPodName, imagePull.Time+1, "", "")
	running.State = &ctrlv1.InstanceEvent_Running{Running: &ctrlv1.Running{}}
	reportEvents(t, ctx, service, bearer, clusterKey, running)
	require.Empty(t, waitingReason(t, ctx, database, replacementPodName, region.ID))
	require.Equal(t, dbtype.DeploymentsStatusReady, deploymentStatus(t, ctx, database, deployment.ID))
	require.Equal(t, []string{replacementPodName}, instanceNames(t, ctx, database, deployment.ID))
	require.Equal(t, wantFailure, readPodFailure(t, ctx, database, deployment.ID))

	reportEmptyDeploymentSnapshot(t, ctx, service, bearer, clusterKey, deployment.K8sName)
	require.Equal(t, 0, instanceCount(t, ctx, database, deployment.ID))
	require.Equal(t, dbtype.DeploymentsStatusReady, deploymentStatus(t, ctx, database, deployment.ID))
	require.Equal(t, wantFailure, readPodFailure(t, ctx, database, deployment.ID))
}

func TestReportInstanceEventsReturnsRetryableErrorWhenPodFailureWriteFails(t *testing.T) {
	ctx := context.Background()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	const bearer = "cluster-test-bearer"
	clusterKey := &ctrlv1.ClusterKey{CellId: "cell-error-test", Platform: "kubernetes", Region: "error-region"}
	service := newInstanceEventTestService(t, &podFailureErrorDatabase{Database: database}, bearer)
	_, err = service.Heartbeat(ctx, authenticatedRequest(bearer, &ctrlv1.HeartbeatRequest{Cluster: clusterKey}))
	require.NoError(t, err)

	event := &ctrlv1.InstanceEvent{
		DeploymentId: "deployment", WorkspaceId: "workspace", PodName: "evicted-pod",
		Time: 1_789_000_123_456, Attributes: map[string]string{"pod_phase": "Failed"},
		State: &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{Reason: "Evicted", Message: "node pressure"}},
	}
	_, err = service.ReportInstanceEvents(ctx, authenticatedRequest(bearer, &ctrlv1.ReportInstanceEventsRequest{
		Cluster: clusterKey, Events: []*ctrlv1.InstanceEvent{event},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

type podFailureErrorDatabase struct {
	db.Database
}

func (*podFailureErrorDatabase) RecordDeploymentPodFailure(context.Context, db.RecordDeploymentPodFailureParams) error {
	return errors.New("injected pod failure write error")
}

func newInstanceEventTestService(t *testing.T, database db.Database, bearer string) *Service {
	t.Helper()
	topologyCache, err := cache.New(cache.Config[string, []db.FindDeploymentTopologyMinReplicasRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 10, Resource: "instance_event_test", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(topologyCache.Close)
	service, err := New(Config{
		Database: database, Bearer: bearer, TopologyCache: topologyCache,
		InstanceEvents: batch.NewNoop[schema.InstanceEventV1](),
	})
	require.NoError(t, err)
	t.Cleanup(service.clusterCache.Close)
	t.Cleanup(service.provisionedCerts.Close)
	return service
}

func authenticatedRequest[T any](bearer string, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+bearer)
	return req
}

func instanceEvent(deployment db.Deployment, workspaceID, podName string, observedAt int64, reason, message string) *ctrlv1.InstanceEvent {
	return &ctrlv1.InstanceEvent{
		WorkspaceId: workspaceID, ProjectId: deployment.ProjectID, AppId: deployment.AppID,
		EnvironmentId: deployment.EnvironmentID, DeploymentId: deployment.ID,
		PodUid: podName + "-uid", PodName: podName, Time: observedAt,
		Attributes: map[string]string{"pod_phase": "Failed"},
		State:      &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{Reason: reason, Message: message}},
	}
}

func reportEvents(t *testing.T, ctx context.Context, service *Service, bearer string, clusterKey *ctrlv1.ClusterKey, events ...*ctrlv1.InstanceEvent) {
	t.Helper()
	_, err := service.ReportInstanceEvents(ctx, authenticatedRequest(bearer, &ctrlv1.ReportInstanceEventsRequest{Cluster: clusterKey, Events: events}))
	require.NoError(t, err)
}

func reportDeploymentUpdate(t *testing.T, ctx context.Context, service *Service, bearer string, clusterKey *ctrlv1.ClusterKey, deploymentName, podName string, status ctrlv1.ReportDeploymentStatusRequest_Update_Instance_Status) {
	t.Helper()
	_, err := service.ReportDeploymentStatus(ctx, authenticatedRequest(bearer, &ctrlv1.ReportDeploymentStatusRequest{
		Cluster: clusterKey,
		Change: &ctrlv1.ReportDeploymentStatusRequest_Update_{Update: &ctrlv1.ReportDeploymentStatusRequest_Update{
			K8SName: deploymentName,
			Instances: []*ctrlv1.ReportDeploymentStatusRequest_Update_Instance{{
				K8SName: podName, Address: "127.0.0.1", CpuMillicores: 250, MemoryMib: 256, Status: status,
			}},
		}},
	}))
	require.NoError(t, err)
}

func reportEmptyDeploymentSnapshot(t *testing.T, ctx context.Context, service *Service, bearer string, clusterKey *ctrlv1.ClusterKey, deploymentName string) {
	t.Helper()
	_, err := service.ReportDeploymentStatus(ctx, authenticatedRequest(bearer, &ctrlv1.ReportDeploymentStatusRequest{
		Cluster: clusterKey,
		Change:  &ctrlv1.ReportDeploymentStatusRequest_Update_{Update: &ctrlv1.ReportDeploymentStatusRequest_Update{K8SName: deploymentName}},
	}))
	require.NoError(t, err)
}

func readPodFailure(t *testing.T, ctx context.Context, database db.Database, deploymentID string) podFailure {
	t.Helper()
	var raw []byte
	err := database.RO().QueryRowContext(ctx, "SELECT last_pod_failure FROM deployments WHERE id = ?", deploymentID).Scan(&raw)
	require.NoError(t, err)
	var failure podFailure
	require.NoError(t, json.Unmarshal(raw, &failure))
	return failure
}

func instanceCount(t *testing.T, ctx context.Context, database db.Database, deploymentID string) int {
	t.Helper()
	var count int
	require.NoError(t, database.RO().QueryRowContext(ctx, "SELECT COUNT(*) FROM instances WHERE deployment_id = ?", deploymentID).Scan(&count))
	return count
}

func instanceNames(t *testing.T, ctx context.Context, database db.Database, deploymentID string) []string {
	t.Helper()
	rows, err := database.RO().QueryContext(ctx, "SELECT k8s_name FROM instances WHERE deployment_id = ? ORDER BY k8s_name", deploymentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rows.Close()) })
	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func waitingReason(t *testing.T, ctx context.Context, database db.Database, podName, regionID string) string {
	t.Helper()
	var reason string
	err := database.RO().QueryRowContext(ctx,
		"SELECT COALESCE(JSON_UNQUOTE(JSON_EXTRACT(container_status, '$.waiting.reason')), '') FROM instances WHERE k8s_name = ? AND region_id = ?",
		podName, regionID,
	).Scan(&reason)
	require.NoError(t, err)
	return reason
}

func deploymentStatus(t *testing.T, ctx context.Context, database db.Database, deploymentID string) dbtype.DeploymentsStatus {
	t.Helper()
	var status dbtype.DeploymentsStatus
	require.NoError(t, database.RO().QueryRowContext(ctx, "SELECT status FROM deployments WHERE id = ?", deploymentID).Scan(&status))
	return status
}
