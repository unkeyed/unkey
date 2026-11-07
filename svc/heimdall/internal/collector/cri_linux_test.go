//go:build linux

package collector

import (
	"context"
	"net"
	"path/filepath"
	"strings"
	"testing"

	eventstypes "github.com/containerd/containerd/api/events"
	containersapi "github.com/containerd/containerd/api/services/containers/v1"
	"github.com/containerd/typeurl/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/grpc"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestCRIWatcherIgnoresExecExitBeforeContainerLookup(t *testing.T) {
	event, err := typeurl.MarshalAny(&eventstypes.TaskExit{
		ContainerID: "container-id",
		ID:          "exec-id",
	})
	require.NoError(t, err)

	watcher := &criWatcher{}
	require.NotPanics(t, func() {
		watcher.handle(t.Context(), "/tasks/exit", event)
	})
}

type lifecycleContainerServer struct {
	containersapi.UnimplementedContainersServer
	podUID string
}

func (s *lifecycleContainerServer) Get(_ context.Context, req *containersapi.GetContainerRequest) (*containersapi.GetContainerResponse, error) {
	return &containersapi.GetContainerResponse{Container: &containersapi.Container{ID: req.ID, Labels: map[string]string{
		criPodUIDLabel: s.podUID, criContainerNameLabel: "primary",
	}}}, nil
}

func TestCRIWatcherPreservesIncarnation(t *testing.T) {
	pod := lifecyclePod()
	index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	require.NoError(t, index.Add(pod))
	bp, rows := lifecycleBuffer(t)
	c := New(Config{CH: bp, PodLister: corelisters.NewPodLister(index), NodeName: "node", Collectors: CollectorSetFrom([]string{"disk"})})
	socket := filepath.Join(t.TempDir(), "containerd.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	server := grpc.NewServer()
	containersapi.RegisterContainersServer(server, &lifecycleContainerServer{podUID: string(pod.UID)})
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-done) })
	w, err := newCRIWatcher(socket, c)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, w.Close()) })
	currentID := strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "containerd://")
	for _, id := range []string{uid.New(uid.InstancePrefix), currentID} {
		start, err := typeurl.MarshalAny(&eventstypes.TaskStart{ContainerID: id})
		require.NoError(t, err)
		w.handle(t.Context(), "/tasks/start", start)
		stop, err := typeurl.MarshalAny(&eventstypes.TaskExit{ContainerID: id, ID: id})
		require.NoError(t, err)
		w.handle(t.Context(), "/tasks/exit", stop)
	}
	bp.Close()
	require.Len(t, *rows, 2)
	require.Equal(t, "start", (*rows)[0].EventKind)
	require.Equal(t, "stop", (*rows)[1].EventKind)
	require.EqualValues(t, 2, (*rows)[0].RestartCount)
	require.Equal(t, (*rows)[0].ContainerUID, (*rows)[1].ContainerUID)
}
