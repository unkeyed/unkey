//go:build linux

package collector

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	eventstypes "github.com/containerd/containerd/api/events"
	containersapi "github.com/containerd/containerd/api/services/containers/v1"
	eventsapi "github.com/containerd/containerd/api/services/events/v1"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestCRIWatcherIgnoresExecExitBeforeContainerLookup(t *testing.T) {
	event, err := anypb.New(&eventstypes.TaskExit{
		ContainerID: "container-id",
		ID:          "exec-id",
	})
	require.NoError(t, err)

	watcher := &criWatcher{}
	require.NotPanics(t, func() {
		watcher.handle(t.Context(), "/tasks/exit", event)
	})
}

func TestCRIWatcher_LifecycleAcrossSubscriptions(t *testing.T) {
	containerID := uid.New(uid.TestPrefix)
	podUID := uid.New(uid.TestPrefix)
	server := grpc.NewServer()
	fixture := &containerdFixture{containerID: containerID, podUID: podUID}
	containersapi.RegisterContainersServer(server, fixture)
	eventsapi.RegisterEventsServer(server, fixture)
	socket := filepath.Join(t.TempDir(), "containerd.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		require.NoError(t, <-serveResult)
	})
	handler := &lifecycleHandler{events: make(chan string, 2)}
	watcher, err := newCRIWatcher(socket, handler)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, watcher.Close()) })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- watcher.Run(ctx) }()
	for _, expected := range []string{"start:" + podUID, "exit:" + podUID} {
		select {
		case event := <-handler.events:
			require.Equal(t, expected, event)
		case <-ctx.Done():
			t.Fatal("missing lifecycle event", expected)
		}
	}
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not stop after cancellation")
	}
}

type lifecycleHandler struct{ events chan string }

func (h *lifecycleHandler) OnStart(_ context.Context, podUID string) { h.events <- "start:" + podUID }
func (h *lifecycleHandler) OnExit(_ context.Context, podUID string)  { h.events <- "exit:" + podUID }

type containerdFixture struct {
	containersapi.UnimplementedContainersServer
	eventsapi.UnimplementedEventsServer
	containerID   string
	podUID        string
	subscriptions int
}

func (f *containerdFixture) Get(ctx context.Context, req *containersapi.GetContainerRequest) (*containersapi.GetContainerResponse, error) {
	if values := metadata.ValueFromIncomingContext(ctx, "containerd-namespace"); len(values) != 1 || values[0] != "k8s.io" {
		return nil, status.Error(codes.InvalidArgument, "missing namespace")
	}
	if req.ID != f.containerID {
		return nil, status.Error(codes.NotFound, "container not found")
	}
	return &containersapi.GetContainerResponse{Container: &containersapi.Container{
		ID:     f.containerID,
		Labels: map[string]string{"io.kubernetes.pod.uid": f.podUID},
	}}, nil
}

func (f *containerdFixture) Subscribe(req *eventsapi.SubscribeRequest, stream eventsapi.Events_SubscribeServer) error {
	if values := metadata.ValueFromIncomingContext(stream.Context(), "containerd-namespace"); len(values) != 1 || values[0] != "k8s.io" {
		return status.Error(codes.InvalidArgument, "missing namespace")
	}
	if len(req.Filters) != 2 || req.Filters[0] != `topic=="/tasks/start"` || req.Filters[1] != `topic=="/tasks/exit"` {
		return status.Error(codes.InvalidArgument, "incorrect event filters")
	}
	f.subscriptions++
	if f.subscriptions == 1 {
		event, err := anypb.New(&eventstypes.TaskStart{ContainerID: f.containerID})
		if err != nil {
			return err
		}
		event.TypeUrl = "containerd.events.TaskStart"
		return stream.Send(&eventsapi.Envelope{Topic: "/tasks/start", Event: event})
	}
	event, err := anypb.New(&eventstypes.TaskExit{ContainerID: f.containerID, ID: f.containerID})
	if err != nil {
		return err
	}
	if err := stream.Send(&eventsapi.Envelope{Topic: "/tasks/exit", Event: event}); err != nil {
		return err
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}
