package deployment

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/podstatus"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

type podEvent struct {
	pod       *corev1.Pod
	eventType watch.EventType
}

// runPodEventHandler reports actual state changes back to the control plane in
// real time until ctx is cancelled.
//
// On every krane-managed pod event it finds the owning workload, rebuilds the
// full deployment status, and reports it (deduplicated via fingerprinting).
// Registering the handler replays every cached pod as an add event, so a new
// leader reports the current state of all workloads.
//
// Events are processed concurrently (up to [conc.DefaultConcurrency]) so that
// a slow RPC for one workload does not block reporting for others.
func (c *Controller) runPodEventHandler(ctx context.Context) {
	events := make(chan podEvent)
	send := func(obj any, eventType watch.EventType) {
		if tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
			obj = tombstone.Obj
		}
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			logger.Error("pod informer: unexpected object", "type", fmt.Sprintf("%T", obj))
			return
		}

		select {
		case events <- podEvent{pod: pod, eventType: eventType}:
		case <-ctx.Done():
		}
	}

	registration, err := c.podInformer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { send(obj, watch.Added) },
		UpdateFunc: func(_, obj any) { send(obj, watch.Modified) },
		DeleteFunc: func(obj any) { send(obj, watch.Deleted) },
	})
	if err != nil {
		logger.Error("pod informer: unable to register event handler", "error", err.Error())
		return
	}
	defer func() {
		if err := c.podInformer.RemoveEventHandler(registration); err != nil {
			logger.Error("pod informer: unable to remove event handler", "error", err.Error())
		}
	}()

	sem := conc.NewSem(conc.DefaultConcurrency)
	defer sem.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			sem.Go(ctx, func(ctx context.Context) {
				c.handlePodEvent(ctx, event.pod, event.eventType)
			})
		}
	}
}

// handlePodEvent processes a single pod event: finds the owning
// workload through the pod's ReplicaSet, builds deployment status, and reports
// it if changed.
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

	rs, err := c.replicaSets.ReplicaSets(pod.Namespace).Get(rsName)
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

	w, err := c.workloadForReplicaSet(rs)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "skipped_rs_gone").Inc()
			logger.Info("pod watch: owning deployment not found, skipping", "pod", pod.Name, "replicaSet", rsName)
			return
		}
		metrics.PodWatchEventsTotal.WithLabelValues("deployment", eventTypeLabel, "error").Inc()
		logger.Error("pod watch: unable to resolve workload", "pod", pod.Name, "replicaSet", rsName, "error", err.Error())
		return
	}

	status, err := c.buildDeploymentStatus(w)
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
