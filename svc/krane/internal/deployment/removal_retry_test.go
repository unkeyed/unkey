package deployment

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRemovalRetryRecoversAbsentObjectsAndLostAcknowledgment(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var reports atomic.Int32
		cluster := &testutil.MockClusterClient{
			GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("topology missing"))
			},
			ReportDeploymentStatusFunc: func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
				if reports.Add(1) == 1 {
					return nil, errors.New("acknowledgment lost")
				}
				return &ctrlv1.ReportDeploymentStatusResponse{}, nil
			},
		}
		controller := newDeleteTestController(fake.NewSimpleClientset(deploymentPod("last-pod")), cluster)
		require.NoError(t, controller.DeleteDeployment(ctx, permanentDelete()))
		require.Zero(t, reports.Load())
		done := make(chan struct{})
		go func() {
			controller.runRemovalRetryLoop(ctx)
			close(done)
		}()
		synctest.Wait()
		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, int32(1), reports.Load())
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Equal(t, int32(2), reports.Load())
		cancel()
		<-done
		require.Empty(t, controller.pendingRemovals)
	})
}

func TestRemovalRetryBackoffIsCappedAndRepeatedHintsDoNotResetIt(t *testing.T) {
	controller := newDeleteTestController(fake.NewSimpleClientset(), &testutil.MockClusterClient{})
	controller.trackRemoval(permanentDelete())
	next := controller.pendingRemovals[deleteTestDeploymentID].retryAt
	for _, delay := range []time.Duration{2, 4, 8, 16, 30, 30} {
		controller.trackRemoval(permanentDelete())
		require.Empty(t, controller.dueRemovals(next.Add(-time.Nanosecond)))
		require.Len(t, controller.dueRemovals(next), 1)
		next = next.Add(delay * time.Second)
		require.Equal(t, next, controller.pendingRemovals[deleteTestDeploymentID].retryAt)
	}
}

func TestRemovalRetryWaitsForCanceledReconcileBeforeReturning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		entered, release := make(chan struct{}), make(chan struct{})
		cluster := &testutil.MockClusterClient{
			GetDesiredDeploymentStateFunc: func(ctx context.Context, _ *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
				close(entered)
				<-ctx.Done()
				<-release
				return nil, ctx.Err()
			},
		}
		controller := newDeleteTestController(fake.NewSimpleClientset(), cluster)
		controller.trackRemoval(permanentDelete())
		done := make(chan struct{})
		go func() {
			controller.runRemovalRetryLoop(ctx)
			close(done)
		}()
		<-entered
		cancel()
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("retry loop returned while reconcile was still running")
		default:
		}
		close(release)
		<-done
		require.Empty(t, cluster.ReportDeploymentStatusCalls)
	})
}
