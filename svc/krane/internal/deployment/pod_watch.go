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
//
// The initial watch must succeed for the controller to start. After that the
// goroutine automatically reconnects with jittered backoff (1-5s) when the
// watch disconnects or times out. A fresh Watch() call with no ResourceVersion
// causes the API server to replay ADDED events for every existing matching
// pod, so startup and reconnect states are covered by the watch itself;
// anything the watch somehow drops is caught by the 30s actual-state resync.
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

			metrics.PodWatchReconnectsTotal.WithLabelValues("deployment", "channel_closed").Inc()
			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("pod watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

			var watchErr error
			w, watchErr = c.watchPods(ctx)
			if watchErr != nil {
				metrics.PodWatchReconnectsTotal.WithLabelValues("deployment", "error").Inc()
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
// Events are handled concurrently with bounded parallelism so that a slow
// control plane RPC for one ReplicaSet does not delay reporting for others.
func (c *Controller) drainPodWatch(ctx context.Context, w watch.Interface) {
	sem := conc.NewSem(conc.DefaultConcurrency)

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

			sem.Go(ctx, func(ctx context.Context) {
				c.handlePodEvent(ctx, pod, event.Type)
			})
		}
	}

	sem.Wait()
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
