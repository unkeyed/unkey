package integration

import (
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestGetDesiredDeploymentStateUsesDeploymentIntent(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	key := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(key, bearer)
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: key.Region})
	require.NoError(t, h.DB.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
		ID: deployment.ID, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
	}))
	req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: key, DeploymentId: deployment.ID})
	req.Header().Set("Authorization", "Bearer "+bearer)
	state, err := svc.GetDesiredDeploymentState(ctx, req)
	require.NoError(t, err)
	require.Equal(t, deployment.K8sName, state.Msg.GetDelete().GetK8SName())

	require.NoError(t, h.DB.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
		ID: deployment.ID, DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	}))
	state, err = svc.GetDesiredDeploymentState(ctx, req)
	require.NoError(t, err)
	require.Equal(t, deployment.ID, state.Msg.GetApply().GetDeploymentId())
}

func TestDeploymentIntentMigrationStopsWritingLegacyStatus(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	key := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(key, bearer)
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: key.Region})
	for _, state := range []mysqltype.DeploymentsDesiredState{
		mysqltype.DeploymentsDesiredStateStopped,
		mysqltype.DeploymentsDesiredStateRunning,
	} {
		require.NoError(t, h.DB.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
			ID: deployment.ID, DesiredState: state,
		}))
		var legacyState string
		err := h.DB.RW().QueryRowContext(ctx,
			"SELECT desired_status FROM deployment_topology WHERE deployment_id = ? LIMIT 1", deployment.ID,
		).Scan(&legacyState)
		require.NoError(t, err)
		require.Equal(t, "running", legacyState, "writer retirement must leave the old column untouched")
		req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: key, DeploymentId: deployment.ID})
		req.Header().Set("Authorization", "Bearer "+bearer)
		response, err := svc.GetDesiredDeploymentState(ctx, req)
		require.NoError(t, err)
		if state == mysqltype.DeploymentsDesiredStateStopped {
			require.Equal(t, deployment.K8sName, response.Msg.GetDelete().GetK8SName())
		} else {
			require.Equal(t, deployment.ID, response.Msg.GetApply().GetDeploymentId())
		}
	}
	for _, stop := range []struct {
		name  string
		write func() error
	}{
		{"failure", func() error {
			return h.DB.FailDeploymentAndStop(ctx, db.FailDeploymentAndStopParams{
				ID: deployment.ID, ProgressingStatuses: []mysqltype.DeploymentsStatus{mysqltype.DeploymentsStatusPending},
			})
		}},
		{"environment", func() error {
			return h.DB.StopDeploymentsByEnvironment(ctx, db.StopDeploymentsByEnvironmentParams{EnvironmentID: deployment.EnvironmentID})
		}},
	} {
		t.Run(stop.name, func(t *testing.T) {
			require.NoError(t, h.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{ID: deployment.ID, Status: mysqltype.DeploymentsStatusPending}))
			require.NoError(t, h.DB.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
				ID: deployment.ID, DesiredState: mysqltype.DeploymentsDesiredStateRunning,
			}))
			require.NoError(t, stop.write())
			var legacyState string
			err := h.DB.RW().QueryRowContext(ctx,
				"SELECT desired_status FROM deployment_topology WHERE deployment_id = ? LIMIT 1", deployment.ID,
			).Scan(&legacyState)
			require.NoError(t, err)
			require.Equal(t, "running", legacyState, "writer retirement must leave the old column untouched")
			req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: key, DeploymentId: deployment.ID})
			req.Header().Set("Authorization", "Bearer "+bearer)
			response, err := svc.GetDesiredDeploymentState(ctx, req)
			require.NoError(t, err)
			require.Equal(t, deployment.K8sName, response.Msg.GetDelete().GetK8SName())
		})
	}
}

func TestDeploymentIntentCutoverDoesNotRestartUnsuccessfulDeployments(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	key := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(key, bearer)
	_, handler := ctrlv1connect.NewClusterServiceHandler(svc)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := ctrlv1connect.NewClusterServiceClient(server.Client(), server.URL)
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: key.Region})
	req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: key, DeploymentId: deployment.ID})
	req.Header().Set("Authorization", "Bearer "+bearer)
	for _, test := range []struct {
		status mysqltype.DeploymentsStatus
		apply  bool
	}{
		{mysqltype.DeploymentsStatusFailed, false},
		{mysqltype.DeploymentsStatusCancelled, false},
		{mysqltype.DeploymentsStatusSuperseded, false},
		{mysqltype.DeploymentsStatusSkipped, false},
		{mysqltype.DeploymentsStatusStopped, true},
		{mysqltype.DeploymentsStatusDeploying, true},
		{mysqltype.DeploymentsStatusReady, true},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			require.NoError(t, h.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{ID: deployment.ID, Status: test.status}))
			state, err := client.GetDesiredDeploymentState(ctx, req)
			require.NoError(t, err)
			if test.apply {
				require.Equal(t, deployment.ID, state.Msg.GetApply().GetDeploymentId())
			} else {
				require.Equal(t, deployment.K8sName, state.Msg.GetDelete().GetK8SName())
			}
			allocated, err := h.DB.SumAllocatedResourcesByWorkspaceID(ctx, db.SumAllocatedResourcesByWorkspaceIDParams{
				WorkspaceID: deployment.WorkspaceID, ExcludeDeploymentID: uid.New(uid.DeploymentPrefix),
			})
			require.NoError(t, err)
			if test.apply {
				require.EqualValues(t, 250, allocated.TotalCpuMillicores)
			} else {
				require.Zero(t, allocated.TotalCpuMillicores)
			}
			syncReq := connect.NewRequest(&ctrlv1.SyncDesiredStateRequest{Cluster: key})
			syncReq.Header().Set("Authorization", "Bearer "+bearer)
			stream, err := client.SyncDesiredState(ctx, syncReq)
			require.NoError(t, err)
			found := false
			for stream.Receive() {
				state := stream.Msg().GetDeployment()
				if state.GetApply().GetDeploymentId() == deployment.ID || state.GetDelete().GetK8SName() == deployment.K8sName {
					found = true
				}
			}
			require.NoError(t, stream.Err())
			require.NoError(t, stream.Close())
			require.Equal(t, test.apply, found, "full sync must omit unsuccessful deployment history")
			if !test.apply {
				require.NoError(t, h.DB.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
					ID: deployment.ID, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
				}))
				current, err := h.DB.FindDeploymentById(ctx, deployment.ID)
				require.NoError(t, err)
				require.Equal(t, test.status, current.Status)
				require.NoError(t, h.DB.UpdateDeploymentDesiredState(ctx, db.UpdateDeploymentDesiredStateParams{
					ID: deployment.ID, DesiredState: mysqltype.DeploymentsDesiredStateRunning,
				}))
				state, err := client.GetDesiredDeploymentState(ctx, req)
				require.NoError(t, err)
				require.Equal(t, deployment.ID, state.Msg.GetApply().GetDeploymentId(), "explicit restart must apply before any separate status write")
				current, err = h.DB.FindDeploymentById(ctx, deployment.ID)
				require.NoError(t, err)
				require.Equal(t, mysqltype.DeploymentsStatusDeploying, current.Status)
			}
		})
	}
}

func TestStoppedDeploymentCleanupPreservesTerminalOutcome(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	key := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(key, bearer)
	for _, test := range []struct {
		status mysqltype.DeploymentsStatus
		want   mysqltype.DeploymentsStatus
	}{
		{mysqltype.DeploymentsStatusFailed, mysqltype.DeploymentsStatusFailed},
		{mysqltype.DeploymentsStatusCancelled, mysqltype.DeploymentsStatusCancelled},
		{mysqltype.DeploymentsStatusSuperseded, mysqltype.DeploymentsStatusSuperseded},
		{mysqltype.DeploymentsStatusSkipped, mysqltype.DeploymentsStatusSkipped},
		{mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsStatusStopped},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{
				Region: key.Region, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
			})
			require.NoError(t, h.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{ID: deployment.ID, Status: test.status}))
			req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
				Cluster: key,
				Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
					K8SName: deployment.K8sName,
				}},
			})
			req.Header().Set("Authorization", "Bearer "+bearer)
			_, err := svc.ReportDeploymentStatus(ctx, req)
			require.NoError(t, err)
			current, err := h.DB.FindDeploymentById(ctx, deployment.ID)
			require.NoError(t, err)
			require.Equal(t, test.want, current.Status)
		})
	}
}

func TestDeploymentFailureCleanupBeforeFinalCompensation(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	key := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(key, bearer)
	for _, test := range []struct {
		status mysqltype.DeploymentsStatus
		want   mysqltype.DeploymentsStatus
	}{
		{mysqltype.DeploymentsStatusDeploying, mysqltype.DeploymentsStatusFailed},
		{mysqltype.DeploymentsStatusCancelled, mysqltype.DeploymentsStatusCancelled},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: key.Region})
			require.NoError(t, h.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
				ID: deployment.ID, Status: test.status,
			}))
			stop := db.FailDeploymentAndStopParams{
				ID: deployment.ID, ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
			}
			require.NoError(t, h.DB.FailDeploymentAndStop(ctx, stop))
			req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
				Cluster: key,
				Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
					K8SName: deployment.K8sName,
				}},
			})
			req.Header().Set("Authorization", "Bearer "+bearer)
			_, err := svc.ReportDeploymentStatus(ctx, req)
			require.NoError(t, err)
			current, err := h.DB.FindDeploymentById(ctx, deployment.ID)
			require.NoError(t, err)
			require.Equal(t, test.want, current.Status)
			require.Equal(t, mysqltype.DeploymentsDesiredStateStopped, current.DesiredState)
			require.NoError(t, h.DB.FailDeploymentAndStop(ctx, stop))
			current, err = h.DB.FindDeploymentById(ctx, deployment.ID)
			require.NoError(t, err)
			require.Equal(t, test.want, current.Status)
		})
	}
}
