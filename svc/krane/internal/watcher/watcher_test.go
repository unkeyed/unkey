package watcher

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/deployment"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDispatch_NilEvent(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Version: 1,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil event")
}

func TestDispatch_NilDeploymentState(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Version: 1,
		Event: &ctrlv1.DeploymentChangeEvent_Deployment{
			Deployment: nil,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil deployment state")
}

func TestWaitReconnectCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	require.False(t, waitReconnect(ctx))
	require.Less(t, time.Since(started), 250*time.Millisecond)
}

func TestWatchWaitsForBothLoopsOnCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		drain := make(chan struct{})
		block := func(ctx context.Context) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			calls.Add(1)
			<-ctx.Done()
			<-drain
			return nil, ctx.Err()
		}
		client := &testutil.MockClusterClient{
			WatchDeploymentChangesFunc: func(ctx context.Context, _ *ctrlv1.WatchDeploymentChangesRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
				return block(ctx)
			},
			SyncDesiredStateFunc: func(ctx context.Context, _ *ctrlv1.SyncDesiredStateRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
				return block(ctx)
			},
		}
		w := New(Config{Cluster: client})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		result := make(chan error, 1)
		go func() { result <- w.Watch(ctx) }()
		time.Sleep(6 * time.Second)
		require.Equal(t, int32(2), calls.Load())
		cancel()
		synctest.Wait()
		require.Empty(t, result)
		close(drain)
		synctest.Wait()
		require.NoError(t, <-result)
		time.Sleep(11 * time.Minute)
		require.Equal(t, int32(2), calls.Load())
	})
}

func TestWatchJoinsInFlightDispatchAfterStreamCloses(t *testing.T) {
	server := httptest.NewServer(connect.NewServerStreamHandler("/sync", func(ctx context.Context, req *connect.Request[ctrlv1.SyncDesiredStateRequest], stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
		return stream.Send(&ctrlv1.DeploymentChangeEvent{
			Version: 1,
			Event: &ctrlv1.DeploymentChangeEvent_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{K8SNamespace: "workloads", K8SName: "deployment"},
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)
	streamClient := connect.NewClient[ctrlv1.SyncDesiredStateRequest, ctrlv1.DeploymentChangeEvent](server.Client(), server.URL+"/sync")
	reportStarted := make(chan struct{})
	reportCancelled := make(chan struct{})
	drain := make(chan struct{})
	client := &testutil.MockClusterClient{
		SyncDesiredStateFunc: func(ctx context.Context, req *ctrlv1.SyncDesiredStateRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			return streamClient.CallServerStream(ctx, connect.NewRequest(req))
		},
		WatchDeploymentChangesFunc: func(ctx context.Context, _ *ctrlv1.WatchDeploymentChangesRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
		ReportDeploymentStatusFunc: func(ctx context.Context, _ *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
			close(reportStarted)
			<-ctx.Done()
			close(reportCancelled)
			<-drain
			return nil, ctx.Err()
		},
	}
	w := New(Config{
		Cluster: client,
		Deployments: deployment.New(deployment.Config{
			ClientSet:    fake.NewClientset(),
			Cluster:      client,
			Fingerprints: cache.NewNoopCache[string, string](),
		}),
	})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	t.Cleanup(func() { close(drain) })
	result := make(chan error, 1)
	go func() { result <- w.Watch(ctx) }()
	select {
	case <-reportStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("full sync did not dispatch deletion")
	}
	cancel()
	select {
	case <-reportCancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch did not receive cancellation")
	}
	select {
	case <-result:
		t.Fatal("watch returned before dispatch drained")
	case <-time.After(50 * time.Millisecond):
	}
	drain <- struct{}{}
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not join drained dispatch")
	}
}
