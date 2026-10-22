package integration

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestReportInstanceEventsRecordsCleanExit(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const bearer = "test-bearer"
	clusterKey := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(clusterKey, bearer)

	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: clusterKey.GetRegion(), DesiredState: ""})
	podName := uid.DNS1035(16)
	status := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
		Cluster: clusterKey,
		Change: &ctrlv1.ReportDeploymentStatusRequest_Update_{
			Update: &ctrlv1.ReportDeploymentStatusRequest_Update{
				K8SName: deployment.K8sName,
				Instances: []*ctrlv1.ReportDeploymentStatusRequest_Update_Instance{{
					K8SName:       podName,
					Address:       "10.0.0.1",
					CpuMillicores: 250,
					MemoryMib:     256,
					Status:        ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING,
				}},
			},
		},
	})
	status.Header().Set("Authorization", "Bearer "+bearer)
	_, err := svc.ReportDeploymentStatus(ctx, status)
	require.NoError(t, err)

	report := func(event *ctrlv1.InstanceEvent) {
		event.PodName = podName
		event.DeploymentId = deployment.ID
		req := connect.NewRequest(&ctrlv1.ReportInstanceEventsRequest{Cluster: clusterKey, Events: []*ctrlv1.InstanceEvent{event}})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, reportErr := svc.ReportInstanceEvents(ctx, req)
		require.NoError(t, reportErr)
	}
	instance := func() db.Instance {
		instances, findErr := h.DB.FindInstancesByDeploymentId(ctx, deployment.ID)
		require.NoError(t, findErr)
		require.Len(t, instances, 1)
		return instances[0]
	}

	finishedAt := time.Now().UnixMilli()
	report(&ctrlv1.InstanceEvent{
		RestartCount: 3,
		Time:         finishedAt,
		State:        &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{ExitCode: 0, Signal: 0, Reason: "Completed", Message: ""}},
	})

	exited := instance().ContainerStatus
	require.Equal(t, uint32(3), exited.RestartCount)
	require.NotNil(t, exited.LastTerminationState)
	require.Equal(t, int32(0), exited.LastTerminationState.ExitCode)
	require.Equal(t, "Completed", exited.LastTerminationState.Reason)
	require.Equal(t, finishedAt, exited.LastTerminationState.FinishedAt)

	report(&ctrlv1.InstanceEvent{
		RestartCount: 3,
		Time:         finishedAt + 1,
		State:        &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{Reason: "CrashLoopBackOff", Message: ""}},
	})

	looping := instance().ContainerStatus
	require.NotNil(t, looping.Waiting)
	require.Equal(t, "CrashLoopBackOff", looping.Waiting.Reason)
	require.Equal(t, exited.LastTerminationState, looping.LastTerminationState)
}
