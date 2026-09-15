package watcher

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
)

func TestDispatch_NilEvent(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil event")
}

func TestDispatch_Checkpoint(t *testing.T) {
	w := &Watcher{}
	require.NoError(t, w.dispatch(t.Context(), &ctrlv1.DeploymentChangeEvent{ResumeToken: []byte("checkpoint")}))
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

func TestWatch_RetriesFailedDeliveryAndResetsExpiredToken(t *testing.T) {
	server := &watchServer{tokens: make(chan []byte, 3)}
	_, handler := ctrlv1connect.NewClusterServiceHandler(server)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	w := New(Config{Cluster: ctrl.NewConnectClusterServiceClient(ctrlv1connect.NewClusterServiceClient(httpServer.Client(), httpServer.URL))})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- w.Watch(ctx) }()
	for _, want := range [][]byte{nil, []byte("committed"), nil} {
		select {
		case got := <-server.tokens:
			require.Equal(t, want, got)
		case <-ctx.Done():
			t.Fatal("watch did not reconnect")
		}
	}
	cancel()
	require.NoError(t, <-done)
}

type watchServer struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	connections atomic.Int32
	tokens      chan []byte
}

func (s *watchServer) WatchDeploymentChanges(ctx context.Context, req *connect.Request[ctrlv1.WatchDeploymentChangesRequest], stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
	s.tokens <- req.Msg.GetResumeToken()
	switch s.connections.Add(1) {
	case 1:
		for _, event := range []*ctrlv1.DeploymentChangeEvent{
			{ResumeToken: []byte("committed")},
			{Event: &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: &ctrlv1.DeploymentState{}}},
			{ResumeToken: []byte("must-not-acknowledge")},
		} {
			if err := stream.Send(event); err != nil {
				return err
			}
		}
		return nil
	case 2:
		return connect.NewError(connect.CodeOutOfRange, errors.New("position purged"))
	default:
		<-ctx.Done()
		return ctx.Err()
	}
}

func (s *watchServer) SyncDesiredState(context.Context, *connect.Request[ctrlv1.SyncDesiredStateRequest], *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
	return nil
}
