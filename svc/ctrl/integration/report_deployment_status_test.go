package integration

import (
	"fmt"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestReportDeploymentStatusDeleteIsIdempotentAfterDeploymentRemoval(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	clusterKey := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(clusterKey, bearer)
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: clusterKey.GetRegion(), DesiredState: ""})

	require.NoError(t, h.DB.DeleteDeploymentTopologiesByEnvironmentId(ctx, deployment.EnvironmentID))
	require.NoError(t, h.DB.DeleteDeploymentsByEnvironmentId(ctx, deployment.EnvironmentID))
	desired := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: clusterKey, DeploymentId: deployment.ID})
	desired.Header().Set("Authorization", "Bearer "+bearer)
	_, err := svc.GetDesiredDeploymentState(ctx, desired)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	for range 2 {
		req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
			Cluster: clusterKey,
			Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{
				Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{K8SName: deployment.K8sName},
			},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		res, err := svc.ReportDeploymentStatus(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
	}
	_, err = h.DB.FindDeploymentById(ctx, deployment.ID)
	require.True(t, db.IsNotFound(err))
}

func TestReportDeploymentStatusDeleteOnlyStopsAfterLastRegion(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	clusterKeys := []*ctrlv1.ClusterKey{
		{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)},
		{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)},
	}
	svc := h.ClusterService(clusterKeys[0], bearer)
	h.ClusterService(clusterKeys[1], bearer)
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region: clusterKeys[0].GetRegion(), DesiredState: mysqltype.DeploymentsDesiredStateStopped,
	})
	for _, key := range clusterKeys {
		req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
			Cluster: key,
			Change: &ctrlv1.ReportDeploymentStatusRequest_Update_{
				Update: &ctrlv1.ReportDeploymentStatusRequest_Update{
					K8SName: deployment.K8sName,
					Instances: []*ctrlv1.ReportDeploymentStatusRequest_Update_Instance{{
						K8SName: uid.DNS1035(16), Address: "10.0.0.1:8080",
						Status: ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING,
					}},
				},
			},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, err := svc.ReportDeploymentStatus(ctx, req)
		require.NoError(t, err)
	}

	for i, key := range clusterKeys {
		for range 2 {
			req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
				Cluster: key,
				Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{
					Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{K8SName: deployment.K8sName},
				},
			})
			req.Header().Set("Authorization", "Bearer "+bearer)
			_, err := svc.ReportDeploymentStatus(ctx, req)
			require.NoError(t, err)
		}
		instances, err := h.DB.FindInstancesByDeploymentId(ctx, deployment.ID)
		require.NoError(t, err)
		require.Len(t, instances, len(clusterKeys)-i-1)
		current, err := h.DB.FindDeploymentById(ctx, deployment.ID)
		require.NoError(t, err)
		if i == 0 {
			require.Equal(t, deployment.Status, current.Status)
			region, err := h.DB.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
				Platform: clusterKeys[1].GetPlatform(), Name: clusterKeys[1].GetRegion(),
			})
			require.NoError(t, err)
			require.Equal(t, region.ID, instances[0].RegionID)
		} else {
			require.Equal(t, mysqltype.DeploymentsStatusStopped, current.Status)
		}
	}
}

func TestReportDeploymentStatusConcurrentReportsForKnownInstances(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const (
		bearer          = "test-bearer"
		platform        = "test"
		deployments     = 20
		instancesPerDep = 3
		workers         = 16
		reportsPerWork  = 30
	)
	clusterKey := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: platform, Region: uid.DNS1035(12)}
	svc := h.ClusterService(clusterKey, bearer)

	report := func(k8sName string, podNames []string, status ctrlv1.ReportDeploymentStatusRequest_Update_Instance_Status) error {
		instances := make([]*ctrlv1.ReportDeploymentStatusRequest_Update_Instance, 0, len(podNames))
		for _, podName := range podNames {
			instances = append(instances, &ctrlv1.ReportDeploymentStatusRequest_Update_Instance{
				K8SName:       podName,
				Address:       "10.0.0.1",
				CpuMillicores: 250,
				MemoryMib:     256,
				Status:        status,
			})
		}
		req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
			Cluster: clusterKey,
			Change: &ctrlv1.ReportDeploymentStatusRequest_Update_{
				Update: &ctrlv1.ReportDeploymentStatusRequest_Update{K8SName: k8sName, Instances: instances},
			},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, reportErr := svc.ReportDeploymentStatus(ctx, req)
		return reportErr
	}

	type reportedDeployment struct {
		deployment db.Deployment
		podNames   []string
	}
	reported := make([]reportedDeployment, deployments)
	for i := range reported {
		deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: clusterKey.GetRegion(), DesiredState: ""})
		podNames := make([]string, instancesPerDep)
		for j := range podNames {
			podNames[j] = uid.DNS1035(16)
		}
		require.NoError(t, report(deployment.K8sName, podNames, ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING))
		reported[i] = reportedDeployment{deployment: deployment, podNames: podNames}
	}

	statuses := []ctrlv1.ReportDeploymentStatusRequest_Update_Instance_Status{
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING,
		ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING,
	}
	errs := make(chan error, workers*reportsPerWork)
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for r := range reportsPerWork {
				target := reported[(w*reportsPerWork+r)%deployments]
				if reportErr := report(target.deployment.K8sName, target.podNames, statuses[r%len(statuses)]); reportErr != nil {
					errs <- fmt.Errorf("report for %s: %w", target.deployment.ID, reportErr)
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for reportErr := range errs {
		require.NoError(t, reportErr)
	}

	for _, target := range reported {
		instances, findErr := h.DB.FindInstancesByDeploymentId(ctx, target.deployment.ID)
		require.NoError(t, findErr)
		require.Len(t, instances, instancesPerDep)
	}
}
