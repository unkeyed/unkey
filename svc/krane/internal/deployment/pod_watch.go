package deployment

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runPodWatchLoop watches deployment pods and reports actual state changes back
// to the control plane in real-time.
//
// The watch filters for pods with "managed-by: krane" and "component: deployment"
// labels. On any pod event it finds the owning ReplicaSet, rebuilds the full
// deployment status, and reports it (deduplicated via fingerprinting).
//
// The initial watch must succeed for the controller to start. After that the
// goroutine automatically reconnects with jittered backoff (1-5s) when the
// watch disconnects or times out.
//
// Returns an error if the initial watch setup fails. Once started the goroutine
// runs until the context is cancelled.
func (c *Controller) runPodWatchLoop(ctx context.Context) error {
	// Verify we can establish a watch before returning to the caller.
	w, err := c.watchPods(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			c.drainPodWatch(ctx, w)

			if ctx.Err() != nil {
				return
			}

			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("pod watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

			var watchErr error
			w, watchErr = c.watchPods(ctx)
			if watchErr != nil {
				logger.Error("pod watch: unable to re-establish watch", "error", watchErr.Error())
				continue
			}
		}
	}()

	return nil
}

// watchPods creates a new Kubernetes watch for krane-managed deployment pods.
func (c *Controller) watchPods(ctx context.Context) (watch.Interface, error) {
	return c.clientSet.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentDeployment().
			ToString(),
	})
}

// drainPodWatch processes events from a pod watch until the channel closes.
func (c *Controller) drainPodWatch(ctx context.Context, w watch.Interface) {
	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Error:
			logger.Error("pod watch: error event", "event", event.Object)
		case watch.Bookmark:
		case watch.Added, watch.Modified, watch.Deleted:
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				logger.Error("unable to cast object to pod")
				continue
			}

			logger.Debug("pod watch: event received", "pod", pod.Name, "namespace", pod.Namespace, "type", event.Type, "phase", pod.Status.Phase, "ip", pod.Status.PodIP)

			rsName := owningReplicaSet(pod)
			if rsName == "" {
				logger.Debug("pod watch: pod has no owning replicaset, skipping", "pod", pod.Name)
				continue
			}

			rs, err := c.clientSet.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, rsName, metav1.GetOptions{})
			if err != nil {
				// RS already deleted — resync loop handles orphan cleanup.
				logger.Debug("pod watch: replicaset not found, skipping", "pod", pod.Name, "replicaSet", rsName, "error", err.Error())
				continue
			}

			status, err := c.buildDeploymentStatus(ctx, rs)
			if err != nil {
				logger.Error("pod watch: unable to build status", "error", err.Error(), "replicaSet", rsName)
				continue
			}

			reported, err := c.reportIfChanged(ctx, status)
			if err != nil {
				logger.Error("pod watch: unable to report status", "error", err.Error(), "replicaSet", rsName)
				continue
			}
			if reported {
				logger.Debug("pod watch: reported changed status", "replicaSet", rsName, "pod", pod.Name, "instances", len(status.GetUpdate().GetInstances()))
			} else {
				logger.Debug("pod watch: status unchanged, skipped report", "replicaSet", rsName, "pod", pod.Name)
			}
		}
	}
}

// owningReplicaSet returns the name of the ReplicaSet that owns this pod, or
// empty string if no controller owner reference with Kind "ReplicaSet" exists.
func owningReplicaSet(pod *corev1.Pod) string {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "ReplicaSet" && ref.Controller != nil && *ref.Controller {
			return ref.Name
		}
	}
	return ""
}
