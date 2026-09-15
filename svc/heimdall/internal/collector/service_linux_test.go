//go:build linux

package collector

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eventstypes "github.com/containerd/containerd/api/events"
	containersapi "github.com/containerd/containerd/api/services/containers/v1"
	eventsapi "github.com/containerd/containerd/api/services/events/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func TestCollector_RunWaitsForCRIExitHandler(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "containerd.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)

	lister := &blockedExitPodLister{
		periodicDone: make(chan struct{}),
		exitEntered:  make(chan struct{}),
		releaseExit:  make(chan struct{}),
	}
	releaseExit := sync.OnceFunc(func() { close(lister.releaseExit) })
	event, err := anypb.New(&eventstypes.TaskExit{ContainerID: "app-container"})
	require.NoError(t, err)
	backend := &exitEventServer{periodicDone: lister.periodicDone, event: event}
	server := grpc.NewServer()
	eventsapi.RegisterEventsServer(server, backend)
	containersapi.RegisterContainersServer(server, backend)
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		require.NoError(t, <-serverDone)
	})

	collector := New(Config{PodLister: lister, CRISocket: socket})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = collector.Run(ctx, time.Hour)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		releaseExit()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("collector did not stop after releasing the exit handler")
		}
	})

	select {
	case <-lister.exitEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("CRI exit handler did not start")
	}
	require.Equal(t, int32(2), lister.calls.Load())
	cancel()
	select {
	case <-done:
		t.Fatal("collector returned before the CRI exit handler finished")
	case <-time.After(100 * time.Millisecond):
	}
	releaseExit()
	select {
	case <-done:
		require.ErrorIs(t, runErr, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("collector did not stop after the CRI exit handler finished")
	}
}

type blockedExitPodLister struct {
	corelisters.PodLister
	calls        atomic.Int32
	periodicDone chan struct{}
	exitEntered  chan struct{}
	releaseExit  chan struct{}
}

func (l *blockedExitPodLister) List(labels.Selector) ([]*corev1.Pod, error) {
	if l.calls.Add(1) == 1 {
		close(l.periodicDone)
		return nil, nil
	}
	close(l.exitEntered)
	<-l.releaseExit
	return nil, nil
}

type exitEventServer struct {
	eventsapi.UnimplementedEventsServer
	containersapi.UnimplementedContainersServer
	periodicDone <-chan struct{}
	event        *anypb.Any
}

func (s *exitEventServer) Subscribe(_ *eventsapi.SubscribeRequest, stream eventsapi.Events_SubscribeServer) error {
	select {
	case <-s.periodicDone:
	case <-stream.Context().Done():
		return stream.Context().Err()
	}
	if err := stream.Send(&eventsapi.Envelope{Topic: "/tasks/exit", Event: s.event}); err != nil {
		return err
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

func (*exitEventServer) Get(_ context.Context, req *containersapi.GetContainerRequest) (*containersapi.GetContainerResponse, error) {
	return &containersapi.GetContainerResponse{Container: &containersapi.Container{
		ID: req.GetID(),
		Labels: map[string]string{
			criPodUIDLabel:        "pod-uid",
			criContainerNameLabel: "app",
		},
	}}, nil
}
