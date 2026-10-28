package integration

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeploymentIntentMigrationLegacyWriterUpdatesDeployment(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	const bearer = "test-bearer"
	key := &ctrlv1.ClusterKey{CellId: uid.DNS1035(12), Platform: "test", Region: uid.DNS1035(12)}
	svc := h.ClusterService(key, bearer)
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: key.Region})
	region, err := h.DB.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: key.Platform, Name: key.Region,
	})
	require.NoError(t, err)
	require.NoError(t, h.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		ID: deployment.ID, Status: mysqltype.DeploymentsStatusFailed,
	}))
	for _, test := range []struct {
		legacy db.DeploymentTopologyDesiredStatus
		intent mysqltype.DeploymentsDesiredState
	}{
		{db.DeploymentTopologyDesiredStatusStopped, mysqltype.DeploymentsDesiredStateStopped},
		{db.DeploymentTopologyDesiredStatusRunning, mysqltype.DeploymentsDesiredStateRunning},
	} {
		t.Run(string(test.intent), func(t *testing.T) {
			require.NoError(t, h.DB.UpdateDeploymentTopologyDesiredStatus(ctx, db.UpdateDeploymentTopologyDesiredStatusParams{
				DeploymentID: deployment.ID, RegionID: region.ID, DesiredStatus: test.legacy,
			}))
			current, err := h.DB.FindDeploymentById(ctx, deployment.ID)
			require.NoError(t, err)
			require.Equal(t, test.intent, current.DesiredState)
			req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: key, DeploymentId: deployment.ID})
			req.Header().Set("Authorization", "Bearer "+bearer)
			response, err := svc.GetDesiredDeploymentState(ctx, req)
			require.NoError(t, err)
			if test.intent == mysqltype.DeploymentsDesiredStateStopped {
				require.Equal(t, deployment.K8sName, response.Msg.GetDelete().GetK8SName())
			} else {
				require.Equal(t, deployment.ID, response.Msg.GetApply().GetDeploymentId())
				require.Equal(t, mysqltype.DeploymentsStatusDeploying, current.Status,
					"legacy wake must reset failure before its notification reaches new readers")
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
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region: key.Region, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
	})
	require.NoError(t, h.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		ID: deployment.ID, Status: mysqltype.DeploymentsStatusCancelled,
	}))
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
	require.Equal(t, mysqltype.DeploymentsStatusCancelled, current.Status)
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
