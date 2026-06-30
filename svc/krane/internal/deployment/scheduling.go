package deployment

import (
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// topologyKeyZone is the standard Kubernetes label for availability zones,
	// used for spreading pods across zones for high availability.
	topologyKeyZone = "topology.kubernetes.io/zone"

	// topologyKeyHostname is the standard Kubernetes label for individual nodes,
	// used for spreading pods across nodes so a single node failure can't take
	// out all of a deployment's replicas.
	topologyKeyHostname = "kubernetes.io/hostname"
)

// deploymentTopologySpread returns topology spread constraints that distribute
// customer workload pods across both nodes and availability zones.
//
// Both constraints use maxSkew=1 with WhenUnsatisfiable=ScheduleAnyway, meaning
// the scheduler prefers even distribution but won't block scheduling if the
// topology is imbalanced. This keeps deployments schedulable even in degraded
// cluster states while still achieving node and zone redundancy under normal
// conditions.
//
// The hostname constraint is what prevents replicas from stacking on a single
// node. It selects on deployment ID so each deployment spreads independently of
// others sharing the nodepool. The zone constraint selects all krane deployment
// pods in the namespace, so single-replica deployments still contribute to
// namespace-level AZ spread and Karpenter gets pressure to provision untrusted
// nodes outside the currently crowded AZ.
func deploymentTopologySpread(deploymentID string) []corev1.TopologySpreadConstraint {
	deploymentSelector := &metav1.LabelSelector{
		MatchLabels: labels.New().DeploymentID(deploymentID),
	}
	fleetSelector := &metav1.LabelSelector{
		MatchLabels: labels.New().
			ManagedByKrane().
			ComponentDeployment(),
	}
	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyHostname,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     deploymentSelector,
		},
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     fleetSelector,
		},
	}
}
