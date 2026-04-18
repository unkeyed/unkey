package sentinel

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runPodWatchLoop watches sentinel pods and re-evaluates deployment health
// whenever a pod event fires.
//
// The Deployment-level watch (runActualStateReportLoop) may not fire when a pod
// crashes or enters ErrImagePull because the Deployment status fields don't
// always change. Watching pods directly gives us immediate visibility into
// individual pod failures.
//
// On any pod event, we find the owning Deployment, re-read its status, and
// report health via determineHealth.
//
// The initial watch must succeed for the controller to start. After that the
// goroutine automatically reconnects with jittered backoff (1-5s) when the
// watch disconnects or times out. On every reconnect (and on initial
// startup) it runs [Controller.syncPodsFromList] first so any transitions
// that landed while the watch was down are picked up immediately instead
// of waiting up to 30s for the resync loop.
func (c *Controller) runPodWatchLoop(ctx context.Context) error {
	w, err := c.watchSentinelPods(ctx)
	if err != nil {
		return err
	}

	go func() {
		// Prime from a list immediately so the very first iteration does not
		// depend on a Modified/Added event firing for pods that were already
		// Ready when krane started.
		c.syncPodsFromList(ctx)

		for {
			c.drainSentinelPodWatch(ctx, w)

			if ctx.Err() != nil {
				return
			}

			metrics.PodWatchReconnectsTotal.WithLabelValues("sentinel", "channel_closed").Inc()
			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("sentinel pod watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

			// Close the reconnect gap: any ContainersReady transition that
			// landed during the backoff window is not in the new watch's
			// replay. A direct list picks it up; fingerprint dedup keeps
			// unchanged pods from generating redundant reports.
			c.syncPodsFromList(ctx)

			var watchErr error
			w, watchErr = c.watchSentinelPods(ctx)
			if watchErr != nil {
				metrics.PodWatchReconnectsTotal.WithLabelValues("sentinel", "error").Inc()
				logger.Error("sentinel pod watch: unable to re-establish watch", "error", watchErr.Error())
				continue
			}
		}
	}()

	return nil
}

// syncPodsFromList lists every krane-managed sentinel pod and feeds each
// one through handlePodEvent as a synthesised Modified event. Used on
// startup and after every watch reconnect to close the event-delivery gap
// that would otherwise be paid for by the 30s actual-state resync.
func (c *Controller) syncPodsFromList(ctx context.Context) {
	pods, err := c.clientSet.CoreV1().Pods(NamespaceSentinel).List(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentSentinel().
			ToString(),
	})
	if err != nil {
		logger.Error("sentinel pod watch: initial list failed", "error", err.Error())
		return
	}
	logger.Info("sentinel pod watch: syncing from list", "pods", len(pods.Items))
	for i := range pods.Items {
		c.handlePodEvent(ctx, &pods.Items[i], watch.Modified)
	}
}

// watchSentinelPods creates a new Kubernetes watch for pods in the sentinel namespace.
func (c *Controller) watchSentinelPods(ctx context.Context) (watch.Interface, error) {
	return c.clientSet.CoreV1().Pods(NamespaceSentinel).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentSentinel().
			ToString(),
	})
}

// drainSentinelPodWatch processes events from a sentinel pod watch.
// On any pod change, it finds the owning Deployment and reports its health.
// Events are handled concurrently with bounded parallelism so that a slow
// control plane RPC for one sentinel does not delay reporting for others.
func (c *Controller) drainSentinelPodWatch(ctx context.Context, w watch.Interface) {
	sem := conc.NewSem(conc.DefaultConcurrency)

	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Error:
			logger.Error("sentinel pod watch: error event", "event", event.Object)
		case watch.Bookmark:
		case watch.Added, watch.Modified, watch.Deleted:
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				logger.Error("sentinel pod watch: unable to cast object to pod")
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
// Deployment, reads its status, and reports health if changed.
func (c *Controller) handlePodEvent(ctx context.Context, pod *corev1.Pod, eventType watch.EventType) {
	eventTypeLabel := strings.ToLower(string(eventType))
	// Only observe delivery lag on real watch events, not synthesised
	// ones from syncPodsFromList. The sync path passes MODIFIED with a
	// list snapshot; its LastTransitionTime is an age, not a lag.
	observePodReadyLagOnTransition(pod)
	logger.Info("sentinel pod watch: event received",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"type", eventType,
		"phase", pod.Status.Phase,
		"ip", pod.Status.PodIP,
		"containers_ready", containersReadyStatus(pod),
		"ready_lag_seconds", containersReadyLagSeconds(pod),
	)

	deploymentName := owningDeployment(pod)
	if deploymentName == "" {
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "skipped_no_deployment").Inc()
		return
	}

	deployment, err := c.clientSet.AppsV1().Deployments(NamespaceSentinel).
		Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		// Deployment gone — resync loop handles orphan cleanup.
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "skipped_deployment_gone").Inc()
		logger.Info("sentinel pod watch: deployment not found, skipping",
			"deployment", deploymentName, "pod", pod.Name, "error", err.Error())
		return
	}

	health := determineHealth(deployment)
	sentinelID := deployment.Labels[labels.LabelKeySentinelID]
	status := &ctrlv1.ReportSentinelStatusRequest{
		K8SName:           deployment.Name,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		Health:            health,
		SentinelId:        sentinelID,
		RunningImage:      convergedImage(deployment),
	}

	reported, err := c.reportIfChanged(ctx, status)
	if err != nil {
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "error").Inc()
		logger.Error("sentinel pod watch: unable to report status",
			"deployment", deploymentName, "error", err.Error())
		return
	}
	if reported {
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "reported").Inc()
		logger.Info("sentinel pod watch: reported changed status",
			"deployment", deploymentName, "pod", pod.Name, "health", health, "available_replicas", deployment.Status.AvailableReplicas)
	} else {
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "deduped").Inc()
	}
}

// observePodReadyLagOnTransition records time-since-ContainersReady-True
// as a watch delivery lag sample.
//
// This is a best-effort proxy, not a true transition detector: the pod's
// LastTransitionTime is static, so every time we observe the same pod we
// would re-record the same transition as if it just happened. The caller
// is expected to invoke this only from the real-time watch path, where
// events fire on every status change; subsequent Modified events that
// carry ContainersReady=True still produce small lags that cluster
// around zero for dashboards.
//
// We deliberately do NOT call this from the resync path or the list
// sync. For those paths the LastTransitionTime is an age, not a lag,
// and would flood the histogram with values like 7800s.
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
		metrics.PodWatchDeliveryLagSeconds.WithLabelValues("sentinel", "watch").Observe(lag)
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

// owningDeployment traverses a pod's owner references to find the name of the
// Deployment that ultimately owns it (Pod → ReplicaSet → Deployment).
// Returns empty string if the ownership chain can't be determined.
func owningDeployment(pod *corev1.Pod) string {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "ReplicaSet" && ref.Controller != nil && *ref.Controller {
			// ReplicaSet name format: "{deployment-name}-{pod-template-hash}"
			// Strip the hash suffix to recover the Deployment name.
			rsName := ref.Name
			for i := len(rsName) - 1; i >= 0; i-- {
				if rsName[i] == '-' {
					return rsName[:i]
				}
			}
			return rsName
		}
	}
	return ""
}
