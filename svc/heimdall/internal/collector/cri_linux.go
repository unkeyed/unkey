//go:build linux

package collector

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	eventstypes "github.com/containerd/containerd/api/events"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/typeurl/v2"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/internal/metrics"
)

// Labels set by the kubelet on each containerd container. We match on these
// to map a task event back to the pod we want to bill. Values are stable
// across containerd versions (mirrors of internal/cri/types/labels.go which
// is not importable).
const (
	criPodUIDLabel        = "io.kubernetes.pod.uid"
	criContainerNameLabel = "io.kubernetes.container.name"
	criPodSandbox         = "POD" // kubelet's label value for pause/infra containers
)

// criEventKind identifies which handler to invoke for a given event.
type criEventKind int

const (
	criStart criEventKind = iota
	criExit
)

// criHandler is the callback contract between the watcher and the collector.
// A nil handler disables that event kind.
type criHandler interface {
	OnStart(ctx context.Context, podUID string)
	OnExit(ctx context.Context, podUID string)
}

// criWatcher subscribes to containerd TaskStart and TaskExit events.
//
// TaskStart matters because without it, a container whose only checkpoint is
// its exit has max(counter) == min(counter) → billing = 0. We emit a baseline
// checkpoint at birth so max-min captures the full lifetime usage.
//
// TaskExit matters because the pod-level cgroup can be torn down within one
// checkpoint_interval of termination, losing the final counter delta.
type criWatcher struct {
	client  *containerd.Client
	handler criHandler
}

// newCRIWatcher dials containerd at socketPath (e.g. /run/containerd/containerd.sock).
// The returned watcher must be Closed on shutdown.
func newCRIWatcher(socketPath string, handler criHandler) (*criWatcher, error) {
	c, err := containerd.New(socketPath, containerd.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return nil, fmt.Errorf("dial containerd at %s: %w", socketPath, err)
	}

	return &criWatcher{client: c, handler: handler}, nil
}

// Run subscribes to containerd events and dispatches them to the handler.
// If the subscription breaks (containerd restart, broken pipe), Run reconnects
// with exponential backoff until ctx is cancelled.
func (w *criWatcher) Run(ctx context.Context) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		err := w.runOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Clean EOF or expected closure: reset backoff and loop promptly.
		if err == nil || errors.Is(err, io.EOF) {
			backoff = time.Second
			continue
		}

		metrics.CRIReconnects.Inc()
		logger.Warn("cri event stream dropped, will reconnect",
			"error", err.Error(),
			"backoff", backoff.String(),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// runOnce drives the event loop until the subscription dies. Callers loop on
// this for reconnect behavior. Uses a child context so the subscription is
// torn down server-side when we return — the parent ctx would otherwise
// leave the gRPC stream alive on the containerd side until the process exits.
func (w *criWatcher) runOnce(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	envCh, errCh := w.client.Subscribe(ctx,
		`topic=="/tasks/start"`,
		`topic=="/tasks/exit"`,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-errCh:
			if err == nil {
				return io.EOF
			}

			return fmt.Errorf("containerd event stream: %w", err)

		case env := <-envCh:
			if env == nil {
				// Channel closed without an error signal.
				return io.EOF
			}

			w.handle(ctx, env.Topic, env.Event)
		}
	}
}

// handle decodes an event and resolves it to a pod UID before dispatching.
// Shared between start/exit because both event types carry a ContainerID.
func (w *criWatcher) handle(ctx context.Context, topic string, raw typeurl.Any) {
	if raw == nil {
		return
	}

	v, err := typeurl.UnmarshalAny(raw)
	if err != nil {
		logger.Warn("failed to decode task event", "topic", topic, "error", err.Error())
		return
	}

	var kind criEventKind
	var containerID string

	switch evt := v.(type) {
	case *eventstypes.TaskStart:
		kind = criStart
		containerID = evt.GetContainerID()
		metrics.CRIEventsReceived.WithLabelValues("start").Inc()
	case *eventstypes.TaskExit:
		kind = criExit
		containerID = evt.GetContainerID()
		metrics.CRIEventsReceived.WithLabelValues("exit").Inc()
	default:
		return
	}

	if containerID == "" {
		return
	}

	podUID, ok := w.resolvePodUID(ctx, containerID)
	if !ok {
		return
	}

	switch kind {
	case criStart:
		w.handler.OnStart(ctx, podUID)
	case criExit:
		w.handler.OnExit(ctx, podUID)
	}
}

// resolvePodUID loads the container from containerd and extracts the pod UID
// from its labels. Returns false if the container is gone, is the pause/infra
// container, or lacks the expected kubelet labels.
func (w *criWatcher) resolvePodUID(ctx context.Context, containerID string) (string, bool) {
	container, err := w.client.LoadContainer(ctx, containerID)
	if err != nil {
		// Container is already gone (common on exit events after kubelet GC).
		return "", false
	}

	labels, err := container.Labels(ctx)
	if err != nil {
		return "", false
	}

	// Ignore the pause/infra container. The primary container's own Task
	// event fires separately and is what we bill.
	if labels[criContainerNameLabel] == criPodSandbox {
		return "", false
	}

	podUID := labels[criPodUIDLabel]
	if podUID == "" {
		return "", false
	}

	return podUID, true
}

func (w *criWatcher) Close() error {
	return w.client.Close()
}
