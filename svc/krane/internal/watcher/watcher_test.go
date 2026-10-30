package watcher

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"connectrpc.com/connect"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/deployment"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDispatch_NilEvent(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil event")
}

func TestDispatch_NilDeploymentState(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Event: &ctrlv1.DeploymentChangeEvent_Deployment{
			Deployment: nil,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil deployment state")
}

func TestWatch_RetriesFailedDeliveryAndResetsUnusableToken(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeInvalidArgument, connect.CodeOutOfRange} {
		t.Run(code.String(), func(t *testing.T) {
			server := &watchServer{tokens: make(chan []byte, 3), code: code}
			_, handler := ctrlv1connect.NewClusterServiceHandler(server)
			httpServer := httptest.NewServer(handler)
			t.Cleanup(httpServer.Close)
			w := New(Config{Cluster: ctrl.NewConnectClusterServiceClient(ctrlv1connect.NewClusterServiceClient(httpServer.Client(), httpServer.URL))})
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- w.Watch(ctx) }()
			// The failed event moves to retries, so the checkpoint after it holds.
			for _, want := range [][]byte{nil, []byte("after-failure"), nil} {
				select {
				case got := <-server.tokens:
					require.Equal(t, want, got)
				case <-ctx.Done():
					t.Fatal("watch did not reconnect")
				}
			}
			cancel()
			require.NoError(t, <-done)
		})
	}
}

type watchServer struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	connections atomic.Int32
	tokens      chan []byte
	code        connect.Code
}

func (s *watchServer) WatchDeploymentChanges(ctx context.Context, req *connect.Request[ctrlv1.WatchDeploymentChangesRequest], stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
	s.tokens <- req.Msg.GetResumeToken()
	switch s.connections.Add(1) {
	case 1:
		for _, event := range []*ctrlv1.DeploymentChangeEvent{
			{ResumeToken: []byte("committed")},
			{Event: &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: &ctrlv1.DeploymentState{}}},
			{ResumeToken: []byte("after-failure")},
		} {
			if err := stream.Send(event); err != nil {
				return err
			}
		}
		return nil
	case 2:
		return connect.NewError(s.code, errors.New("unusable resume token"))
	default:
		<-ctx.Done()
		return ctx.Err()
	}
}

func (s *watchServer) SyncDesiredState(context.Context, *connect.Request[ctrlv1.SyncDesiredStateRequest], *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
	return nil
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

// One workload that fails on every apply must not hold back later changes in
// the region, and its own change must apply once the failure clears.
func TestWatchAppliesLaterEventsWhileFailedEventRetries(t *testing.T) {
	deleteEvent := func(name string) *ctrlv1.DeploymentChangeEvent {
		return &ctrlv1.DeploymentChangeEvent{
			Event: &ctrlv1.DeploymentChangeEvent_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{K8SNamespace: "workloads", K8SName: name},
					},
				},
			},
		}
	}
	server := httptest.NewServer(connect.NewServerStreamHandler("/watch", func(ctx context.Context, _ *connect.Request[ctrlv1.WatchDeploymentChangesRequest], stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
		for _, event := range []*ctrlv1.DeploymentChangeEvent{deleteEvent("broken"), deleteEvent("healthy")} {
			if err := stream.Send(event); err != nil {
				return err
			}
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	t.Cleanup(server.Close)
	watchClient := connect.NewClient[ctrlv1.WatchDeploymentChangesRequest, ctrlv1.DeploymentChangeEvent](server.Client(), server.URL+"/watch")

	var brokenAttempts atomic.Int32
	reported := make(chan string, 2)
	client := &testutil.MockClusterClient{
		WatchDeploymentChangesFunc: func(ctx context.Context, req *ctrlv1.WatchDeploymentChangesRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			return watchClient.CallServerStream(ctx, connect.NewRequest(req))
		},
		SyncDesiredStateFunc: func(ctx context.Context, _ *ctrlv1.SyncDesiredStateRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			return nil, errors.New("full sync unavailable")
		},
		ReportDeploymentStatusFunc: func(_ context.Context, req *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
			name := req.GetDelete().GetK8SName()
			if name == "broken" && brokenAttempts.Add(1) == 1 {
				return nil, errors.New("rejected")
			}
			reported <- name
			return &ctrlv1.ReportDeploymentStatusResponse{}, nil
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
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	result := make(chan error, 1)
	go func() { result <- w.Watch(ctx) }()

	for _, want := range []string{"healthy", "broken"} {
		select {
		case got := <-reported:
			require.Equal(t, want, got)
		case <-ctx.Done():
			t.Fatalf("%s was never applied", want)
		}
	}
	require.Equal(t, int32(2), brokenAttempts.Load())

	cancel()
	require.NoError(t, <-result)
}

// A deleted deployment produces no stream event, so a pending apply must not
// outlive it and recreate the workload once the failure clears.
func TestWatchDropsPendingApplyForDeletedDeployment(t *testing.T) {
	server := httptest.NewServer(connect.NewServerStreamHandler("/watch", func(ctx context.Context, _ *connect.Request[ctrlv1.WatchDeploymentChangesRequest], stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
		if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
			Event: &ctrlv1.DeploymentChangeEvent_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{DeploymentId: "deployment_gone", K8SNamespace: "workloads", K8SName: "gone"},
					},
				},
			},
		}); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	t.Cleanup(server.Close)
	watchClient := connect.NewClient[ctrlv1.WatchDeploymentChangesRequest, ctrlv1.DeploymentChangeEvent](server.Client(), server.URL+"/watch")

	lookups := make(chan string, 4)
	client := &testutil.MockClusterClient{
		WatchDeploymentChangesFunc: func(ctx context.Context, req *ctrlv1.WatchDeploymentChangesRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			return watchClient.CallServerStream(ctx, connect.NewRequest(req))
		},
		SyncDesiredStateFunc: func(ctx context.Context, _ *ctrlv1.SyncDesiredStateRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
			return nil, errors.New("full sync unavailable")
		},
		GetDesiredDeploymentStateFunc: func(_ context.Context, req *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
			lookups <- req.GetDeploymentId()
			return nil, connect.NewError(connect.CodeNotFound, errors.New("deployment not found"))
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
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	result := make(chan error, 1)
	go func() { result <- w.Watch(ctx) }()

	select {
	case id := <-lookups:
		require.Equal(t, "deployment_gone", id)
	case <-ctx.Done():
		t.Fatal("pending apply was never retried")
	}
	require.Eventually(t, func() bool {
		var pending dto.Metric
		require.NoError(t, metrics.StreamRetriesPending.Write(&pending))
		return pending.GetGauge().GetValue() == 0
	}, 5*time.Second, 50*time.Millisecond)

	cancel()
	require.NoError(t, <-result)
}
