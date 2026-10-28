package integration

import (
	"fmt"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

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
