package sentinel

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
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
// watch disconnects or times out. A fresh Watch() call with no ResourceVersion
// causes the API server to replay ADDED events for every existing matching
// pod, so startup and reconnect states are covered by the watch itself;
// anything the watch somehow drops is caught by the 30s actual-state resync.
func (c *Controller) runPodWatchLoop(ctx context.Context) error {
	w, err := c.watchSentinelPods(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			c.drainSentinelPodWatch(ctx, w)

			if ctx.Err() != nil {
				return
			}

			metrics.PodWatchReconnectsTotal.WithLabelValues("sentinel", "channel_closed").Inc()
			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("sentinel pod watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

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
	c.lagRecorder.Observe(ctx, pod, eventType)
	logger.Info("sentinel pod watch: event received",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"type", eventType,
		"phase", pod.Status.Phase,
		"ip", pod.Status.PodIP,
		"containers_ready", podstatus.ReadyStatus(pod),
		"ready_lag_seconds", podstatus.ReadyLagSeconds(pod),
	)

	deploymentName := owningDeployment(pod)
	if deploymentName == "" {
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "skipped_no_deployment").Inc()
		return
	}

	deployment, err := c.clientSet.AppsV1().Deployments(NamespaceSentinel).
		Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Deployment gone — resync loop handles orphan cleanup.
			metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "skipped_deployment_gone").Inc()
			logger.Info("sentinel pod watch: deployment not found, skipping",
				"deployment", deploymentName, "pod", pod.Name)
			return
		}
		metrics.PodWatchEventsTotal.WithLabelValues("sentinel", eventTypeLabel, "error").Inc()
		logger.Error("sentinel pod watch: deployment get failed",
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
