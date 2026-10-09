package deployment

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	rolloutGateInterval = 15 * time.Second

	stalledRolloutRetryAfter = 6 * time.Hour
	stalledRolloutHaltWindow = time.Hour

	rolloutSlotAnnotation            = "unkey.com/rollout-slot"
	rolloutStalledTemplateAnnotation = "unkey.com/rollout-stalled-template"
	rolloutStalledAtAnnotation       = "unkey.com/rollout-stalled-at"

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
		c.gateRollouts(ctx, time.Now())
	})
}

func (c *Controller) gateRollouts(ctx context.Context, now time.Time) {
	var deployments []appsv1.Deployment
	err := c.forEachDeploymentPage(ctx, func(page []appsv1.Deployment) {
		deployments = append(deployments, page...)
	})
	if err != nil {
		logger.Error("rollout gate: unable to list deployments", "error", err.Error())
		return
	}

	plan := planRollouts(deployments, c.maxConcurrentRollouts, now)
	metrics.RolloutGateWorkloads.WithLabelValues("rolling").Set(float64(plan.slotsHeld))
	metrics.RolloutGateWorkloads.WithLabelValues("stalled").Set(float64(plan.stalled + len(plan.stall)))
	metrics.RolloutGateWorkloads.WithLabelValues("waiting").Set(float64(plan.waiting))
	if plan.halted {
		logger.Warn("rollout gate: halted, too many recent stalled rollouts", "recent_stalled", plan.recentStalled+len(plan.stall), "waiting", plan.waiting)
	}

	for _, d := range plan.stall {
		logger.Warn("rollout gate: rollout stalled, pausing", "namespace", d.Namespace, "name", d.Name)
		c.setRolloutState(ctx, d, rolloutState{paused: true, slot: false, stalledTemplate: podTemplateHash(d), stalledAt: now})
	}
	for _, d := range plan.finish {
		c.setRolloutState(ctx, d, rolloutState{paused: true, slot: false, stalledTemplate: "", stalledAt: time.Time{}})
	}
	for _, d := range plan.admit {
		c.setRolloutState(ctx, d, rolloutState{paused: false, slot: true, stalledTemplate: "", stalledAt: time.Time{}})
	}
	for _, d := range plan.admitUnhealthy {
		c.setRolloutState(ctx, d, rolloutState{paused: false, slot: false, stalledTemplate: "", stalledAt: time.Time{}})
	}
}

// rolloutPlan is the set of changes one gate tick makes.
//
// A rollout that exceeds its progress deadline is paused with its slot freed.
// Its old pods keep serving, because maxUnavailable is zero. It rolls again
// when krane renders a different template, or after stalledRolloutRetryAfter.
// While maxConcurrent rollouts stalled within stalledRolloutHaltWindow, the
// gate admits nothing, so a broken pod template stops after maxConcurrent
// workloads per window. The window is shorter than the retry delay, so a
// workload that stalls on every retry cannot halt the gate for good.
//
// Waiting workloads that already miss available pods roll without a slot, at
// most maxConcurrent per tick. A broken customer image would otherwise hold a
// slot until it stalls.
type rolloutPlan struct {
	stall          []*appsv1.Deployment
	finish         []*appsv1.Deployment
	admit          []*appsv1.Deployment
	admitUnhealthy []*appsv1.Deployment

	slotsHeld     int
	stalled       int
	recentStalled int
	waiting       int
	halted        bool
}

func planRollouts(deployments []appsv1.Deployment, maxConcurrent int, now time.Time) rolloutPlan {
	var plan rolloutPlan
	var waitingHealthy, waitingUnhealthy []*appsv1.Deployment
	for i := range deployments {
		d := &deployments[i]
		switch {
		case !d.Spec.Paused:
			if holdsRolloutSlot(d) {
				plan.slotsHeld++
			}
			switch {
			case holdsRolloutSlot(d) && rolloutStalled(d):
				plan.stall = append(plan.stall, d)
			case rolloutComplete(d):
				plan.finish = append(plan.finish, d)
			}
		case stalledWithin(d, now, stalledRolloutRetryAfter):
			plan.stalled++
			if stalledWithin(d, now, stalledRolloutHaltWindow) {
				plan.recentStalled++
			}
		case !rolloutWaiting(d):
		case d.Status.AvailableReplicas >= desiredReplicas(d):
			waitingHealthy = append(waitingHealthy, d)
		default:
			waitingUnhealthy = append(waitingUnhealthy, d)
		}
	}

	plan.halted = plan.recentStalled+len(plan.stall) >= max(maxConcurrent, 1)
	plan.waiting = len(waitingHealthy) + len(waitingUnhealthy)
	if plan.halted {
		return plan
	}

	sortOldestFirst(waitingHealthy)
	sortOldestFirst(waitingUnhealthy)
	free := max(maxConcurrent-plan.slotsHeld, 0)
	plan.admit = waitingHealthy[:min(free, len(waitingHealthy))]
	plan.admitUnhealthy = waitingUnhealthy[:min(max(maxConcurrent, 0), len(waitingUnhealthy))]
	plan.waiting -= len(plan.admit) + len(plan.admitUnhealthy)

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

// stalledWithin reports whether the current template of d stalled within
// window before now.
func stalledWithin(d *appsv1.Deployment, now time.Time, window time.Duration) bool {
	template := podTemplateHash(d)
	if template == "" || d.Annotations[rolloutStalledTemplateAnnotation] != template {
		return false
	}

	stalledAt, err := time.Parse(time.RFC3339, d.Annotations[rolloutStalledAtAnnotation])
	if err != nil {
		return false
	}

	return now.Sub(stalledAt) < window
}

// podTemplateHash identifies the pod template of d. It returns "" if the
// template cannot be encoded, which never matches a recorded stall.
func podTemplateHash(d *appsv1.Deployment) string {
	encoded, err := json.Marshal(d.Spec.Template)
	if err != nil {
		logger.Error("rollout gate: unable to encode pod template", "namespace", d.Namespace, "name", d.Name, "error", err.Error())
		return ""
	}
	return hash.Sha256(string(encoded))
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

// rolloutState is the gate-owned state of a Deployment. Zero fields remove
// their annotation.
type rolloutState struct {
	paused          bool
	slot            bool
	stalledTemplate string
	stalledAt       time.Time
}

// setRolloutState writes state to d. The patch carries the listed
// resourceVersion, so it fails if d changed since the gate read it; the next
// tick decides again.
func (c *Controller) setRolloutState(ctx context.Context, d *appsv1.Deployment, state rolloutState) {
	annotations := map[string]any{
		rolloutSlotAnnotation:            nil,
		rolloutStalledTemplateAnnotation: nil,
		rolloutStalledAtAnnotation:       nil,
	}
	if state.slot {
		annotations[rolloutSlotAnnotation] = "true"
	}
	if state.stalledTemplate != "" {
		annotations[rolloutStalledTemplateAnnotation] = state.stalledTemplate
		annotations[rolloutStalledAtAnnotation] = state.stalledAt.UTC().Format(time.RFC3339)
	}

	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"resourceVersion": d.ResourceVersion,
			"annotations":     annotations,
		},
		"spec": map[string]any{"paused": state.paused},
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
			"paused", state.paused,
			"error", err.Error(),
		)
		return
	}

	logger.Info("rollout gate: updated deployment", "namespace", d.Namespace, "name", d.Name, "paused", state.paused, "slot", state.slot)
}
