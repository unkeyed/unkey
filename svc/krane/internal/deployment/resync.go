package deployment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// runActualStateResyncLoop periodically reports actual instance state to the
// control plane for every deployment ReplicaSet.
//
// This is a fast, lightweight safety net that complements [Controller.runPodWatchLoop].
// The watch handles real-time events, but can miss updates during network partitions,
// restarts, or buffer overflows. This loop catches the drift by rebuilding and
// reporting status for every RS every 30 seconds.
//
// This loop does NOT fetch or apply desired state — that is handled independently
// by [Controller.runDesiredStateResyncLoop] so that slow control plane RPCs cannot
// delay instance reporting.
func (c *Controller) runActualStateResyncLoop(ctx context.Context) {
	c.runResyncLoop(ctx, 30*time.Second, func() {
		logger.Info("running actual state resync")
		c.forEachReplicaSet(ctx, func(ctx context.Context, rs *appsv1.ReplicaSet) {
			status, err := c.buildDeploymentStatus(ctx, rs)
			if err != nil {
				logger.Error("actual state resync: unable to build deployment status", "error", err.Error(), "replicaSet", rs.Name)
				return
			}
			reported, err := c.reportIfChanged(ctx, status)
			if err != nil {
				logger.Error("actual state resync: unable to report deployment status", "error", err.Error(), "replicaSet", rs.Name)
				return
			}
			if reported {
				// Resync found drift the watch didn't deliver. This is the
				// "pod watch missed an event" smoking-gun signal — a
				// healthy cluster should see this counter stay flat.
				metrics.ResyncCorrectionsTotal.WithLabelValues("deployment").Inc()
				logger.Info("actual state resync: reported changed deployment status", "replicaSet", rs.Name)
			}
		})
	})
}

// runDesiredStateResyncLoop periodically reconciles every deployment ReplicaSet
// against the control plane's desired state.
//
// This is a consistency safety net that complements the streaming desired state
// channel. It runs every minute, fetching the desired state for each RS and
// applying or deleting as needed. Because this involves potentially slow RPCs
// (GetDesiredDeploymentState), it runs independently from actual state reporting
// so it cannot delay instance updates.
func (c *Controller) runDesiredStateResyncLoop(ctx context.Context) {
	c.runResyncLoop(ctx, time.Minute, func() {
		logger.Info("running desired state resync")
		var owners sync.Map
		c.forEachReplicaSet(ctx, func(ctx context.Context, rs *appsv1.ReplicaSet) {
			owners.Store(rs.Namespace+"/"+rs.Name, struct{}{})
			c.reconcileDesiredState(ctx, rs)
		})
		c.forEachDeploymentPod(ctx, func(ctx context.Context, pod *corev1.Pod) {
			if _, exists := owners.Load(pod.Namespace + "/" + owningReplicaSet(pod)); exists {
				return
			}
			c.reconcileOrphanPod(ctx, pod)
		})
	})
}

func (c *Controller) runResyncLoop(ctx context.Context, interval time.Duration, resync func()) {
	resync()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resync()
		}
	}
}

func (c *Controller) forEachDeploymentPod(ctx context.Context, fn func(context.Context, *corev1.Pod)) {
	cursor := ""
	for {
		pods, err := c.clientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: labels.New().ManagedByKrane().ComponentDeployment().ToString(),
			Limit:         500,
			Continue:      cursor,
		})
		if err != nil {
			logger.Error("unable to list deployment pods", "error", err.Error())
			return
		}
		conc.ForEach(ctx, pods.Items, fn)
		cursor = pods.Continue
		if cursor == "" {
			return
		}
	}
}

func (c *Controller) reconcileOrphanPod(ctx context.Context, pod *corev1.Pod) {
	replicaSetName := owningReplicaSet(pod)
	if replicaSetName != "" {
		if _, err := c.clientSet.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, replicaSetName, metav1.GetOptions{}); err == nil {
			return
		} else if !apierrors.IsNotFound(err) {
			logger.Error("unable to inspect pod owner", "error", err.Error(), "pod", pod.Name, "replicaSet", replicaSetName)
			return
		}
	}

	deploymentID, ok := labels.GetDeploymentID(pod.Labels)
	if !ok {
		logger.Error("unable to get deployment ID from orphan pod", "pod", pod.Name)
		return
	}
	if err := c.ReconcileDeployment(ctx, &ctrlv1.DeleteDeployment{
		DeploymentId: deploymentID,
		K8SNamespace: pod.Namespace,
		K8SName:      replicaSetName,
	}); err != nil {
		logger.Error("unable to reconcile orphan pod", "error", err.Error(), "deployment_id", deploymentID, "pod", pod.Name)
	}
}

// forEachReplicaSet paginates through all krane-managed deployment ReplicaSets
// and calls fn for each one concurrently.
func (c *Controller) forEachReplicaSet(ctx context.Context, fn func(ctx context.Context, rs *appsv1.ReplicaSet)) {
	cursor := ""
	for {
		replicaSets, err := c.clientSet.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{
			LabelSelector: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				ToString(),
			Limit:    500,
			Continue: cursor,
		})
		if err != nil {
			logger.Error("unable to list replicaSets", "error", err.Error())
			return
		}

		conc.ForEach(ctx, replicaSets.Items, fn)

		cursor = replicaSets.Continue
		if cursor == "" {
			break
		}
	}
}

// reconcileDesiredState fetches the desired state for a single ReplicaSet from
// the control plane and applies or deletes as needed.
func (c *Controller) reconcileDesiredState(ctx context.Context, replicaSet *appsv1.ReplicaSet) {
	deploymentID, ok := labels.GetDeploymentID(replicaSet.Labels)
	if !ok {
		logger.Error("unable to get deployment ID", "replicaSet", replicaSet.Name)
		return
	}

	if err := c.ReconcileDeployment(ctx, &ctrlv1.DeleteDeployment{
		DeploymentId: deploymentID,
		K8SNamespace: replicaSet.GetNamespace(),
		K8SName:      replicaSet.GetName(),
	}); err != nil {
		logger.Error("unable to reconcile deployment", "error", err.Error(), "deployment_id", deploymentID)
	}
}

func (c *Controller) ReconcileDeployment(ctx context.Context, hint *ctrlv1.DeleteDeployment) error {
	if hint.GetDeploymentId() == "" {
		return fmt.Errorf("deployment ID is required")
	}

	unlock := c.reconcileLocks.Lock(hint.GetDeploymentId())
	defer unlock()

	res, err := c.cluster.GetDesiredDeploymentState(ctx, &ctrlv1.GetDesiredDeploymentStateRequest{
		Cluster:      c.clusterKey(),
		DeploymentId: hint.GetDeploymentId(),
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			hint.Permanent = true
			return c.DeleteDeployment(ctx, hint)
		}
		return fmt.Errorf("unable to get desired deployment state: %w", err)
	}

	switch state := res.GetState().(type) {
	case *ctrlv1.DeploymentState_Apply:
		c.forgetRemoval(hint.GetDeploymentId())
		return c.ApplyDeployment(ctx, state.Apply)
	case *ctrlv1.DeploymentState_Delete:
		return c.DeleteDeployment(ctx, state.Delete)
	default:
		return fmt.Errorf("unhandled desired deployment state type %T", state)
	}
}
