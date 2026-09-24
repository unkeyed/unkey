package deployment

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/podstatus"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
// Events are processed concurrently (up to [maxPodWatchConcurrency]) so that a
// slow RPC for one ReplicaSet does not block reporting for others.
func (c *Controller) runPodWatchLoop(ctx context.Context) {
	for ctx.Err() == nil {
		w, err := c.watchPods(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			metrics.PodWatchReconnectsTotal.WithLabelValues("deployment", "error").Inc()
			logger.Error("pod watch: unable to establish watch", "error", err.Error())

			if !waitPodWatchBackoff(ctx) {
				return
			}
			continue
		}

		c.drainPodWatch(ctx, w)
		if ctx.Err() != nil {
			return
		}

		metrics.PodWatchReconnectsTotal.WithLabelValues("deployment", "channel_closed").Inc()
		logger.Warn("pod watch: disconnected, reconnecting")
		if !waitPodWatchBackoff(ctx) {
			return
		}
	}
}

func waitPodWatchBackoff(ctx context.Context) bool {
	backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
// Events are handled concurrently with bounded parallelism so that a slow
// control plane RPC for one ReplicaSet does not delay reporting for others.
func (c *Controller) drainPodWatch(ctx context.Context, w watch.Interface) {
	sem := conc.NewSem(conc.DefaultConcurrency)
	defer func() {
		w.Stop()
		sem.Wait()
	}()

	for {
		var event watch.Event
		var ok bool
		select {
		case <-ctx.Done():
			return
		case event, ok = <-w.ResultChan():
			if !ok {
				return
			}
		}

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

			sem.Go(ctx, func(ctx context.Context) {
				c.handlePodEvent(ctx, pod, event.Type)
			})
		}
	}
}

// handlePodEvent processes a single pod watch event: finds the owning
// ReplicaSet, builds deployment status, and reports it if changed.
func (c *Controller) handlePodEvent(ctx context.Context, pod *corev1.Pod, eventType watch.EventType) {
	eventTypeLabel := strings.ToLower(string(eventType))
	c.lagRecorder.Observe(ctx, pod, eventType)
	logger.Info("pod watch: event received",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"type", eventType,
		"phase", pod.Status.Phase,
		"ip", pod.Status.PodIP,
		"containers_ready", podstatus.ReadyStatus(pod),
		"ready_lag_seconds", podstatus.ReadyLagSeconds(pod),
	)

	// Capture per-container lifecycle events independent of the coarse
	// status report below. Runs before any early-return path so a missing
	// ReplicaSet, transient RS-get failure, or buildDeploymentStatus error
	// doesn't suppress exit/crashloop events the dashboard needs to show.
	// Best-effort: errors are logged inside, never returned.
	c.reportInstanceEvents(ctx, pod)

	rsName := owningReplicaSet(pod)
	if rsName == "" {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "skipped_no_rs").Inc()
		logger.Info("pod watch: pod has no owning replicaset, skipping", "pod", pod.Name)
		return
	}

	rs, err := c.clientSet.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, rsName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// RS already deleted — resync loop handles orphan cleanup.
			metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "skipped_rs_gone").Inc()
			logger.Info("pod watch: replicaset not found, skipping", "pod", pod.Name, "replicaSet", rsName)
			return
		}
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "error").Inc()
		logger.Error("pod watch: replicaset get failed", "pod", pod.Name, "replicaSet", rsName, "error", err.Error())
		return
	}

	status, err := c.buildDeploymentStatus(ctx, rs)
	if err != nil {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "error").Inc()
		logger.Error("pod watch: unable to build status", "error", err.Error(), "replicaSet", rsName)
		return
	}

	reported, err := c.reportIfChanged(ctx, status)
	if err != nil {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "error").Inc()
		logger.Error("pod watch: unable to report status", "error", err.Error(), "replicaSet", rsName)
		return
	}
	if reported {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "reported").Inc()
		logger.Info("pod watch: reported changed status", "replicaSet", rsName, "pod", pod.Name, "instances", len(status.GetUpdate().GetInstances()))
	} else {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "deduped").Inc()
		logger.Info("pod watch: status unchanged, skipped report", "replicaSet", rsName, "pod", pod.Name)
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
