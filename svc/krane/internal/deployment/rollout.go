package deployment

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	rolloutGateInterval = 15 * time.Second

	rolloutSlotAnnotation = "unkey.com/rollout-slot"

	// progressDeadlineExceededReason is the Progressing condition reason the
	// Deployment controller sets when a rollout makes no progress for
	// spec.progressDeadlineSeconds.
	progressDeadlineExceededReason = "ProgressDeadlineExceeded"
)

// runRolloutGateLoop limits how many workloads roll out at the same time.
//
// Workload Deployments are paused at rest. A template change on a paused
// Deployment creates no ReplicaSet, so the change waits. The gate unpauses
// waiting Deployments while fewer than maxConcurrentRollouts healthy workloads
// roll, and pauses each Deployment again when its rollout completes. A paused
// Deployment still follows its HPA. All gate state is on the Deployments, so a
// new leader continues where the previous one stopped.
func (c *Controller) runRolloutGateLoop(ctx context.Context) {
	c.runResyncLoop(ctx, rolloutGateInterval, func() {
		c.gateRollouts(ctx)
	})
}

func (c *Controller) gateRollouts(ctx context.Context) {
	var deployments []appsv1.Deployment
	err := c.forEachDeploymentPage(ctx, func(page []appsv1.Deployment) {
		deployments = append(deployments, page...)
	})
	if err != nil {
		logger.Error("rollout gate: unable to list deployments", "error", err.Error())
		return
	}

	plan := planRollouts(deployments, c.maxConcurrentRollouts)
	metrics.RolloutGateWorkloads.WithLabelValues("rolling").Set(float64(plan.slotsHeld))
	metrics.RolloutGateWorkloads.WithLabelValues("stalled").Set(float64(plan.stalled))
	metrics.RolloutGateWorkloads.WithLabelValues("waiting").Set(float64(plan.waiting))

	for _, d := range plan.finish {
		c.setRolloutState(ctx, d, true, false)
	}
	for _, d := range plan.admit {
		c.setRolloutState(ctx, d, false, true)
	}
	for _, d := range plan.admitUnhealthy {
		c.setRolloutState(ctx, d, false, false)
	}
}

type rolloutPlan struct {
	finish         []*appsv1.Deployment
	admit          []*appsv1.Deployment
	admitUnhealthy []*appsv1.Deployment

	slotsHeld int
	stalled   int
	waiting   int
}

func planRollouts(deployments []appsv1.Deployment, maxConcurrent int) rolloutPlan {
	var plan rolloutPlan
	var waitingHealthy, waitingUnhealthy []*appsv1.Deployment
	for i := range deployments {
		d := &deployments[i]
		switch {
		case !d.Spec.Paused:
			if holdsRolloutSlot(d) {
				plan.slotsHeld++
				if rolloutStalled(d) {
					plan.stalled++
				}
			}
			if rolloutComplete(d) {
				plan.finish = append(plan.finish, d)
			}
		case !rolloutWaiting(d):
		case d.Status.AvailableReplicas >= desiredReplicas(d):
			waitingHealthy = append(waitingHealthy, d)
		default:
			waitingUnhealthy = append(waitingUnhealthy, d)
		}
	}

	sortOldestFirst(waitingHealthy)
	sortOldestFirst(waitingUnhealthy)
	free := max(maxConcurrent-plan.slotsHeld, 0)
	plan.admit = waitingHealthy[:min(free, len(waitingHealthy))]
	plan.admitUnhealthy = waitingUnhealthy[:min(max(maxConcurrent, 0), len(waitingUnhealthy))]
	plan.waiting = len(waitingHealthy) + len(waitingUnhealthy) - len(plan.admit) - len(plan.admitUnhealthy)

	return plan
}

func desiredReplicas(d *appsv1.Deployment) int32 {
	if d.Spec.Replicas == nil {
		return 1
	}
	return *d.Spec.Replicas
}

func holdsRolloutSlot(d *appsv1.Deployment) bool {
	return d.Annotations[rolloutSlotAnnotation] == "true"
}

func rolloutWaiting(d *appsv1.Deployment) bool {
	return d.Status.ObservedGeneration >= d.Generation && d.Status.UpdatedReplicas < d.Status.Replicas
}

func rolloutComplete(d *appsv1.Deployment) bool {
	want := desiredReplicas(d)
	s := d.Status
	return s.ObservedGeneration >= d.Generation &&
		s.UpdatedReplicas == want &&
		s.Replicas == want &&
		s.AvailableReplicas == want
}

func rolloutStalled(d *appsv1.Deployment) bool {
	for _, cond := range d.Status.Conditions {
		if cond.Type == appsv1.DeploymentProgressing {
			return cond.Status == corev1.ConditionFalse && cond.Reason == progressDeadlineExceededReason
		}
	}
	return false
}

func sortOldestFirst(deployments []*appsv1.Deployment) {
	slices.SortFunc(deployments, func(a, b *appsv1.Deployment) int {
		return cmp.Or(
			a.CreationTimestamp.Time.Compare(b.CreationTimestamp.Time),
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Name, b.Name),
		)
	})
}

// setRolloutState pauses or unpauses d and sets or removes its slot
// annotation. The patch carries the listed resourceVersion, so it fails if d
// changed since the gate read it; the next tick decides again.
func (c *Controller) setRolloutState(ctx context.Context, d *appsv1.Deployment, paused, slot bool) {
	var slotValue any
	if slot {
		slotValue = "true"
	}

	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"resourceVersion": d.ResourceVersion,
			"annotations":     map[string]any{rolloutSlotAnnotation: slotValue},
		},
		"spec": map[string]any{"paused": paused},
	})
	if err != nil {
		logger.Error("rollout gate: unable to marshal patch", "error", err.Error())
		return
	}

	_, err = c.clientSet.AppsV1().Deployments(d.Namespace).Patch(ctx, d.Name, types.MergePatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerRolloutGate,
	})
	if err != nil {
		logger.Warn("rollout gate: unable to update deployment",
			"namespace", d.Namespace,
			"name", d.Name,
			"paused", paused,
			"error", err.Error(),
		)
		return
	}

	logger.Info("rollout gate: updated deployment", "namespace", d.Namespace, "name", d.Name, "paused", paused, "slot", slot)
}
