package sentinel

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	appsv1 "k8s.io/api/apps/v1"
)

// determineHealth reports whether the sentinel is serving traffic.
//
// Health is decoupled from rollout success — it answers "are pods serving?"
// not "did the latest deploy converge?". During a failed rolling update, old
// pods stay healthy and serving, so frontline keeps routing to them.
// Rollout convergence is tracked separately via [convergedImage].
func determineHealth(d *appsv1.Deployment) ctrlv1.Health {
	desiredReplicas := int32(0)
	if d.Spec.Replicas != nil {
		desiredReplicas = *d.Spec.Replicas
	}

	if desiredReplicas == 0 {
		return ctrlv1.Health_HEALTH_PAUSED
	}

	if d.Status.AvailableReplicas > 0 {
		return ctrlv1.Health_HEALTH_HEALTHY
	}

	return ctrlv1.Health_HEALTH_UNHEALTHY
}

// convergedImage returns the image fully rolled out across all replicas, or
// empty string if the rollout is not converged.
//
// The control plane uses this to detect when the desired image is actually
// running. Returning empty during a rolling update prevents the NotifyReady
// gate from firing before the rollout completes.
func convergedImage(d *appsv1.Deployment) string {
	desiredReplicas := int32(0)
	if d.Spec.Replicas != nil {
		desiredReplicas = *d.Spec.Replicas
	}

	// Not converged if:
	//   - k8s hasn't observed the latest spec (status fields are stale)
	//   - not all replicas on the new spec
	//   - fewer than desired are available
	//   - any are still unavailable
	if d.Status.ObservedGeneration < d.Generation ||
		d.Status.UpdatedReplicas < desiredReplicas ||
		d.Status.AvailableReplicas < desiredReplicas ||
		d.Status.UnavailableReplicas > 0 {
		return ""
	}

	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == "sentinel" {
			return c.Image
		}
	}
	if len(d.Spec.Template.Spec.Containers) > 0 {
		return d.Spec.Template.Spec.Containers[0].Image
	}
	return ""
}
