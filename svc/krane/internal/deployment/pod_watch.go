package deployment

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
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
// Events are processed concurrently (up to [maxPodWatchConcurrency]) so that a
// slow RPC for one ReplicaSet does not block reporting for others.
//
// The initial watch must succeed for the controller to start. After that the
// goroutine automatically reconnects with jittered backoff (1-5s) when the
// watch disconnects or times out. On every reconnect (and on initial
// startup) it runs [Controller.syncPodsFromList] first so any transitions
// that landed while the watch was down are picked up immediately instead
// of waiting up to 30s for the resync loop.
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
		// Prime from a list immediately so the very first iteration does not
		// depend on a Modified/Added event firing for pods that were already
		// Ready when krane started.
		c.syncPodsFromList(ctx)

		for {
			c.drainPodWatch(ctx, w)

			if ctx.Err() != nil {
				return
			}

			metrics.PodWatchReconnectsTotal.WithLabelValues("deployment", "channel_closed").Inc()
			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("pod watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

			// Close the reconnect gap: any ContainersReady transition that
			// landed during the backoff window is not in the new watch's
			// replay. A direct list picks it up; fingerprint dedup keeps
			// unchanged pods from generating redundant reports.
			c.syncPodsFromList(ctx)

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

// syncPodsFromList lists every krane-managed deployment pod and feeds each
// one through handlePodEvent as a synthesised Modified event. Used on
// startup and after every watch reconnect to close the event-delivery gap
// that would otherwise be paid for by the 30s actual-state resync.
func (c *Controller) syncPodsFromList(ctx context.Context) {
	pods, err := c.clientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentDeployment().
			ToString(),
	})
	if err != nil {
		logger.Error("pod watch: initial list failed", "error", err.Error())
		return
	}
	logger.Info("pod watch: syncing from list", "pods", len(pods.Items))
	for i := range pods.Items {
		c.handlePodEvent(ctx, &pods.Items[i], watch.Modified)
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
	// Only observe delivery lag on real watch events, not synthesised
	// ones from syncPodsFromList. [watch.EventType] values for a true
	// watch are ADDED/MODIFIED/DELETED from the API server; the sync
	// path passes MODIFIED with a list snapshot, which would skew the
	// histogram upward with stale "lags".
	observePodReadyLagOnTransition(pod)
	logger.Info("pod watch: event received",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"type", eventType,
		"phase", pod.Status.Phase,
		"ip", pod.Status.PodIP,
		"containers_ready", containersReadyStatus(pod),
		"ready_lag_seconds", containersReadyLagSeconds(pod),
	)

	rsName := owningReplicaSet(pod)
	if rsName == "" {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "skipped_no_rs").Inc()
		logger.Info("pod watch: pod has no owning replicaset, skipping", "pod", pod.Name)
		return
	}

	rs, err := c.clientSet.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, rsName, metav1.GetOptions{})
	if err != nil {
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "skipped_rs_gone").Inc()
		// RS already deleted — resync loop handles orphan cleanup.
		logger.Info("pod watch: replicaset not found, skipping", "pod", pod.Name, "replicaSet", rsName, "error", err.Error())
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

// observePodReadyLagOnTransition records time-since-ContainersReady-True
// as a watch delivery lag sample.
//
// This is a best-effort proxy, not a true transition detector: the pod's
// LastTransitionTime is static, so every time we observe the same pod we
// would re-record the same transition as if it just happened. The caller
// is expected to invoke this only from the real-time watch path
// ([Controller.handlePodEvent] on an actual API server event), where
// events fire on every status change; subsequent Modified events that
// carry ContainersReady=True still produce small lags that cluster
// around zero for dashboards.
//
// We deliberately do NOT call this from the resync path or the list
// sync. For those paths the LastTransitionTime is an age, not a lag,
// and would flood the histogram with values like 7800s. The drift
// signal from resync is captured via
// [metrics.ResyncCorrectionsTotal] instead.
func observePodReadyLagOnTransition(pod *corev1.Pod) {
	if pod == nil {
		return
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady || cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.LastTransitionTime.IsZero() {
			return
		}
		lag := time.Since(cond.LastTransitionTime.Time).Seconds()
		if lag < 0 {
			return
		}
		metrics.PodWatchDeliveryLagSeconds.WithLabelValues("deployment", "watch").Observe(lag)
		return
	}
}

// containersReadyStatus returns the pod's ContainersReady condition as a
// short string ("true"/"false"/"unknown"/"absent") for log lines.
func containersReadyStatus(pod *corev1.Pod) string {
	if pod == nil {
		return "absent"
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady {
			continue
		}
		switch cond.Status {
		case corev1.ConditionTrue:
			return "true"
		case corev1.ConditionFalse:
			return "false"
		case corev1.ConditionUnknown:
			return "unknown"
		default:
			return "unknown"
		}
	}
	return "absent"
}

// containersReadyLagSeconds returns the time since kubelet flipped
// ContainersReady=True on the pod, or -1 when unknown. Used as a log field
// so tailing krane shows how fresh each watch event is relative to the
// actual transition.
func containersReadyLagSeconds(pod *corev1.Pod) float64 {
	if pod == nil {
		return -1
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady || cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.LastTransitionTime.IsZero() {
			return -1
		}
		return time.Since(cond.LastTransitionTime.Time).Seconds()
	}
	return -1
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
